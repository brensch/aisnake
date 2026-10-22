//! Decode Battlesnake replay GIFs (exporter render, 224x224, 11x11 board) into game states.
//! A port of decode2.py: simulates the game from frame 0 and checks every turn against the
//! painted cells. Usage: snakedecode <gif dir> <out.jsonl.zst> <summary.tsv>

use rayon::prelude::*;
use std::collections::HashSet;
use std::fs::File;
use std::io::{BufWriter, Write};
use std::path::PathBuf;

const PITCH: usize = 20;
const ORIGIN: usize = 3;
const N: usize = 11;
const FOOD: Rgb = (255, 92, 117);
const STEPS: [(i32, i32); 4] = [(1, 0), (-1, 0), (0, 1), (0, -1)];

type Rgb = (u8, u8, u8);
type Cell = (usize, usize);

fn near(a: Rgb, b: Rgb, tol: i32) -> bool {
    (a.0 as i32 - b.0 as i32).abs() + (a.1 as i32 - b.1 as i32).abs() + (a.2 as i32 - b.2 as i32).abs() <= tol
}

fn painted(c: Rgb) -> bool {
    let (r, g, b) = c;
    if c == (240, 240, 240) || c == (255, 255, 255) || c == (239, 239, 240) || (r > 225 && g > 225 && b > 225) {
        return false;
    }
    let (mx, mn) = (r.max(g).max(b), r.min(g).min(b));
    // the faint "Battlesnake" logo: low-saturation lilac (colour spread <= 58 in every
    // render seen), so pastel snakes such as pink #f0a0f0 still count as paint
    // ...bluish (blue above red) and never brighter than ~218 red, unlike pastel pinks
    // such as #f5b4f0 or #f7bef9
    let lilac = b > g && r > g && b > r && r <= 225 && b >= 150 && (mx - mn) < 65 && !near(c, FOOD, 40);
    !lilac
}

fn dist(a: Rgb, b: Rgb) -> i32 {
    (a.0 as i32 - b.0 as i32).abs() + (a.1 as i32 - b.1 as i32).abs() + (a.2 as i32 - b.2 as i32).abs()
}

/// Is `c` this snake's paint: within tolerance of its colour, and nearer to it than to
/// any other live snake's (so similar colours, e.g. slate and indigo, stay apart)?
fn belongs(c: Rgb, own: Rgb, others: &[Rgb]) -> bool {
    let d = dist(c, own);
    d <= 90 && others.iter().all(|&o| dist(c, o) >= d)
}

/// The exporter's dead-snake grey, ~#cdcdcd. Live snakes can be grey too (#888, #999...),
/// so only this narrow band counts as a dead body.
fn grey(c: Rgb) -> bool {
    let (mx, mn) = (c.0.max(c.1).max(c.2), c.0.min(c.1).min(c.2));
    mx - mn < 10 && (195..=216).contains(&c.0)
}

/// Counts in first-seen order; ties go to the earliest (as Python's Counter.most_common).
#[derive(Default, Clone)]
struct Votes(Vec<(Rgb, u32)>);
impl Votes {
    fn add(&mut self, c: Rgb) {
        match self.0.iter_mut().find(|(k, _)| *k == c) {
            Some(e) => e.1 += 1,
            None => self.0.push((c, 1)),
        }
    }
    fn top(&self) -> Rgb {
        let mut best = self.0[0];
        for &e in &self.0[1..] {
            if e.1 > best.1 {
                best = e;
            }
        }
        best.0
    }
}

struct CellInfo {
    samples: [Rgb; 36],
    paint: usize,
    dom: Option<Rgb>,
    /// coloured samples even where a dead snake's grey body dominates (a live head can be
    /// drawn over it)
    raw: usize,
    raw_dom: Option<Rgb>,
    /// bare board: no snake paint and no dead (grey) body either
    blank: bool,
    /// some of a dead snake's grey body shows in this cell
    greyish: bool,
    /// this cell or a neighbour shows grey dead body
    by_grey: bool,
    /// excused last turn and unchanged since: a dead snake's art left on the board
    stale_art: bool,
}

fn read_frame(px: &[u8], w: usize) -> (Vec<CellInfo>, HashSet<Cell>) {
    let at = |x: usize, y: usize| {
        let o = (y * w + x) * 3;
        (px[o], px[o + 1], px[o + 2])
    };
    let mut cells = Vec::with_capacity(N * N);
    let mut food = HashSet::new();
    for i in 0..N {
        for j in 0..N {
            let (x0, y0) = (ORIGIN + PITCH * i, ORIGIN + PITCH * j);
            let mut samples = [(0, 0, 0); 36];
            for a in 0..6 {
                for b in 0..6 {
                    samples[a * 6 + b] = at(x0 + 1 + 3 * a, y0 + 1 + 3 * b);
                }
            }
            let mut votes = Votes::default();
            let (mut paint, mut foods, mut greys, mut ring) = (0, 0, 0, 0);
            for (k, &c) in samples.iter().enumerate() {
                // outer ring of the 6x6 sample grid: a food dot never reaches it, a segment does
                let (a, b) = (k / 6, k % 6);
                if (a == 0 || a == 5 || b == 0 || b == 5) && painted(c) && !grey(c) {
                    ring += 1;
                }
                if painted(c) {
                    // grey is a dead snake's body; it doesn't count as paint
                    if grey(c) {
                        greys += 1;
                    } else {
                        paint += 1;
                        votes.add(c);
                    }
                }
                if near(c, FOOD, 30) {
                    foods += 1;
                }
            }
            let mut dom = if paint > 0 { Some(votes.top()) } else { None };
            let (raw, raw_dom) = (paint, dom);
            // food can sit on a dead snake's grey body: judge it against the coloured paint only
            if foods >= 3 && foods as f64 >= paint as f64 * 0.4 && ring <= 4 {
                food.insert((i, j));
                paint = 0;
                dom = None;
            }
            // a mostly-grey cell is a dead snake's body, whatever art is drawn over it
            if greys >= 13 && greys > paint {
                paint = 0;
                dom = None;
            }
            cells.push(CellInfo { samples, paint, dom, raw, raw_dom, blank: paint == 0 && greys < 3, greyish: greys >= 3, by_grey: false, stale_art: false });
        }
    }
    // a cell touching a dead (grey) body: where a dead snake's coloured end art can sit
    let grey_at: Vec<bool> = cells.iter().map(|c| c.greyish).collect();
    for i in 0..N {
        for j in 0..N {
            let mut g = grey_at[i * N + j];
            for (dx, dy) in STEPS {
                let (x, y) = (i as i32 + dx, j as i32 + dy);
                if x >= 0 && y >= 0 && x < N as i32 && y < N as i32 {
                    g |= grey_at[x as usize * N + y as usize];
                }
            }
            cells[i * N + j].by_grey = g;
        }
    }
    (cells, food)
}

fn changed(a: &[Rgb; 36], b: &[Rgb; 36]) -> usize {
    a.iter().zip(b).filter(|(x, y)| !near(**x, **y, 30)).count()
}

struct Snake {
    body: Vec<Cell>,
    alive: bool,
    votes: Votes,
}

/// Streams GIF frames composited onto an RGB canvas (as browsers and PIL do). Frames stay
/// palette-indexed until copied, and the canvas is reused, so nothing is kept per frame.
struct Frames {
    dec: gif::Decoder<std::io::Cursor<Vec<u8>>>,
    global: Vec<u8>,
    canvas: Vec<u8>,
    saved: Vec<u8>,
    pending: Option<(gif::DisposalMethod, usize, usize, usize, usize)>,
}

impl Frames {
    fn open(bytes: Vec<u8>) -> Result<Self, String> {
        let mut opts = gif::DecodeOptions::new();
        opts.set_color_output(gif::ColorOutput::Indexed);
        let dec = opts.read_info(std::io::Cursor::new(bytes)).map_err(|e| e.to_string())?;
        if dec.width() != 224 || dec.height() != 224 {
            return Err(format!("unexpected size {}x{}", dec.width(), dec.height()));
        }
        let global = dec.global_palette().map(|p| p.to_vec()).unwrap_or_default();
        Ok(Frames { dec, global, canvas: vec![0; 224 * 224 * 3], saved: Vec::new(), pending: None })
    }

    fn next(&mut self) -> Result<Option<&[u8]>, String> {
        const W: usize = 224;
        // apply the previous frame's disposal first
        if let Some((d, fl, ft, fw, fh)) = self.pending.take() {
            match d {
                gif::DisposalMethod::Background => {
                    for y in ft..(ft + fh).min(W) {
                        let row = y * W;
                        self.canvas[(row + fl) * 3..(row + (fl + fw).min(W)) * 3].fill(0);
                    }
                }
                gif::DisposalMethod::Previous => std::mem::swap(&mut self.canvas, &mut self.saved),
                _ => {}
            }
        }
        let Some(f) = self.dec.read_next_frame().map_err(|e| e.to_string())? else {
            return Ok(None);
        };
        if f.dispose == gif::DisposalMethod::Previous {
            self.saved.clone_from(&self.canvas);
        }
        let pal: &[u8] = f.palette.as_deref().unwrap_or(&self.global);
        let (fl, ft, fw, fh) = (f.left as usize, f.top as usize, f.width as usize, f.height as usize);
        let tr = f.transparent;
        for y in 0..fh.min(W.saturating_sub(ft)) {
            let src = &f.buffer[y * fw..y * fw + fw];
            let row = (ft + y) * W;
            for (x, &ix) in src.iter().enumerate().take(W.saturating_sub(fl)) {
                if Some(ix) == tr {
                    continue;
                }
                let (d, p) = ((row + fl + x) * 3, ix as usize * 3);
                if p + 2 < pal.len() {
                    self.canvas[d..d + 3].copy_from_slice(&pal[p..p + 3]);
                }
            }
        }
        self.pending = Some((f.dispose, fl, ft, fw, fh));
        Ok(Some(&self.canvas))
    }
}

struct Game {
    json: String,
    turns: usize,
    snakes: usize,
    errors: usize,
}

/// Each live snake's possible new head cells this turn, best first: (score, cell).
fn candidates(snakes: &[Snake], cells: &[CellInfo], prev: &[CellInfo]) -> Vec<Vec<(usize, Cell)>> {
    let idx = |c: Cell| c.0 * N + c.1;
    // cells that stay occupied by a live snake's body next turn (all but its tail)
    let blocked: HashSet<Cell> =
        snakes.iter().filter(|s| s.alive).flat_map(|s| s.body[..s.body.len() - 1].iter().copied()).collect();
    snakes
        .iter()
        .map(|s| {
            let mut out = Vec::new();
            if !s.alive {
                return out;
            }
            let (hx, hy) = (s.body[0].0 as i32, s.body[0].1 as i32);
            let col = s.votes.top();
            let others: Vec<Rgb> =
                snakes.iter().filter(|o| o.alive && !std::ptr::eq(*o, s)).map(|o| o.votes.top()).collect();
            let mut weak = Vec::new();
            for (dx, dy) in STEPS {
                let (x, y) = (hx + dx, hy + dy);
                if x < 0 || y < 0 || x >= N as i32 || y >= N as i32 {
                    continue;
                }
                let c = (x as usize, y as usize);
                if blocked.contains(&c) {
                    continue;
                }
                let ci = &cells[idx(c)];
                let d = changed(&ci.samples, &prev[idx(c)].samples);
                let (p, dom) = (ci.raw, ci.raw_dom);
                let same = dom.is_some_and(|dc| belongs(dc, col, &others));
                let score = if p >= 13 { 1000 } else { 0 } + if same { 500 } else { 0 } + d + p;
                if p >= 13 || (p >= 4 && d >= 4) {
                    out.push((score, c));
                } else if d + p > 0 {
                    // faint head art: kept as a last resort (a live snake must move somewhere)
                    weak.push((score, c));
                }
            }
            out.sort_by(|a, b| b.cmp(a));
            if out.is_empty() {
                weak.sort_by(|a, b| b.cmp(a));
                out = weak;
            }
            out
        })
        .collect()
}

/// The bodies the snakes would have after these moves (None = eliminated).
fn apply(snakes: &[Snake], moved: &[Option<Cell>], prev_food: &HashSet<Cell>) -> Vec<Option<Vec<Cell>>> {
    snakes
        .iter()
        .zip(moved)
        .map(|(s, m)| {
            let best = (*m)?;
            if !s.alive {
                return None;
            }
            let mut body = Vec::with_capacity(s.body.len() + 1);
            body.push(best);
            body.extend_from_slice(&s.body[..s.body.len() - 1]);
            // Battlesnake: move (the tail follows), then eating stacks a copy of the tail
            if prev_food.contains(&best) {
                body.push(*body.last().unwrap());
            }
            Some(body)
        })
        .collect()
}

/// Record a dead snake's cells: its ends may keep their coloured art, the rest turns grey.
fn mark_ghost(ghosts: &mut [u8; N * N], body: &[Cell]) {
    for &c in body {
        let g = &mut ghosts[c.0 * N + c.1];
        *g = (*g).max(1);
    }
    for c in [body[0], *body.last().unwrap()] {
        ghosts[c.0 * N + c.1] = 2;
    }
    // the fatal move: the render shows the dead snake one step on, head art and all
    let (hx, hy) = (body[0].0 as i32, body[0].1 as i32);
    for (dx, dy) in STEPS {
        let (x, y) = (hx + dx, hy + dy);
        if x >= 0 && y >= 0 && x < N as i32 && y < N as i32 {
            ghosts[x as usize * N + y as usize] = 2;
        }
    }
}

/// A painted cell nobody covers is fine if a dead snake left it: its grey body, or coloured
/// end art attached to that grey body. A wrongly killed snake shows no grey, so stays an error.
fn excused(ghost: u8, ci: &CellInfo) -> bool {
    // partial paint is leftover art; a solid segment (34-36 of 36) would be a live snake
    ghost > 0 && (ci.by_grey || ci.stale_art || ci.paint <= 30)
}

/// Cells where these bodies disagree with the picture: painted but uncovered, covered
/// but empty, or a middle segment painted another snake's colour.
fn mismatch(bodies: &[Option<Vec<Cell>>], snakes: &[Snake], cells: &[CellInfo], ghosts: &[u8; N * N]) -> usize {
    let idx = |c: Cell| c.0 * N + c.1;
    let mut cover = [false; N * N];
    let mut ends = [false; N * N];
    for b in bodies.iter().flatten() {
        for &c in b {
            cover[idx(c)] = true;
        }
        ends[idx(b[0])] = true;
        ends[idx(*b.last().unwrap())] = true;
    }
    let mut bad = cells
        .iter()
        .zip(cover)
        .zip(ghosts)
        .zip(ends)
        .filter(|(((ci, cov), ghost), end)| (ci.paint >= 20 && !cov && !excused(**ghost, ci)) || (ci.blank && *cov && !end))
        .count();
    for (b, s) in bodies.iter().zip(snakes) {
        if let Some(b) = b {
            let col = s.votes.top();
            let others: Vec<Rgb> = snakes
                .iter()
                .zip(bodies)
                .filter(|(o, ob)| ob.is_some() && !std::ptr::eq(*o, s))
                .map(|(o, _)| o.votes.top())
                .collect();
            for &c in &b[1..b.len() - 1] {
                let ci = &cells[idx(c)];
                if c != b[0] && c != *b.last().unwrap() && ci.paint >= 20 && ci.dom.is_some_and(|dc| !belongs(dc, col, &others)) {
                    bad += 1;
                }
            }
        }
    }
    bad
}

/// Pick each live snake's move: best-first greedy; if that turn doesn't match the picture,
/// try every combination of candidate moves (or elimination) and keep the best match.
fn assign(
    snakes: &[Snake],
    cells: &[CellInfo],
    prev: &[CellInfo],
    prev_food: &HashSet<Cell>,
    ghosts: &[u8; N * N],
) -> Vec<Option<Cell>> {
    let cands = candidates(snakes, cells, prev);
    let mut options: Vec<(usize, usize, Cell)> =
        cands.iter().enumerate().flat_map(|(sid, cs)| cs.iter().map(move |&(sc, c)| (sc, sid, c))).collect();
    options.sort_by(|a, b| b.cmp(a));
    let mut greedy: Vec<Option<Cell>> = vec![None; snakes.len()];
    let mut taken: HashSet<Cell> = HashSet::new();
    for (_, sid, c) in options {
        if greedy[sid].is_none() && !taken.contains(&c) {
            greedy[sid] = Some(c);
            taken.insert(c);
        }
    }
    let base = mismatch(&apply(snakes, &greedy, prev_food), snakes, cells, ghosts);
    if base == 0 {
        return greedy;
    }
    // search: each live snake takes one of its candidates or is eliminated
    let live: Vec<usize> = (0..snakes.len()).filter(|&i| snakes[i].alive).collect();
    let choices: Vec<Vec<Option<(usize, Cell)>>> =
        live.iter()
            .map(|&i| if cands[i].is_empty() { vec![None] } else { cands[i].iter().map(|&x| Some(x)).collect() })
            .collect();
    let total: usize = choices.iter().map(|c| c.len()).product();
    if total > 20_000 {
        return greedy;
    }
    let (mut best, mut best_key) = (greedy.clone(), (base, usize::MAX - greedy_score(&cands, &greedy)));
    let mut pick = vec![0usize; live.len()];
    for _ in 0..total {
        let mut moved: Vec<Option<Cell>> = vec![None; snakes.len()];
        let mut used = HashSet::new();
        let mut ok = true;
        let mut score = 0;
        for (k, &sid) in live.iter().enumerate() {
            if let Some((sc, c)) = choices[k][pick[k]] {
                ok &= used.insert(c);
                score += sc;
                moved[sid] = Some(c);
            }
        }
        if ok {
            let key = (mismatch(&apply(snakes, &moved, prev_food), snakes, cells, ghosts), usize::MAX - score);
            if key < best_key {
                best_key = key;
                best = moved;
            }
        }
        for k in 0..pick.len() {
            pick[k] += 1;
            if pick[k] < choices[k].len() {
                break;
            }
            pick[k] = 0;
        }
    }
    best
}

fn greedy_score(cands: &[Vec<(usize, Cell)>], moved: &[Option<Cell>]) -> usize {
    moved
        .iter()
        .zip(cands)
        .filter_map(|(m, cs)| m.and_then(|c| cs.iter().find(|x| x.1 == c).map(|x| x.0)))
        .sum()
}

fn decode(bytes: Vec<u8>, id: &str) -> Result<Game, String> {
    let mut fr = Frames::open(bytes)?;
    let mut snakes: Vec<Snake> = Vec::new();
    let mut turns = Vec::new();
    let mut errors = 0usize;
    let mut owned: Vec<(usize, Rgb, Vec<usize>)> = Vec::new();
    let mut prev: Option<Vec<CellInfo>> = None;
    let mut prev_food: HashSet<Cell> = HashSet::new();
    let idx = |c: Cell| c.0 * N + c.1;
    let mut ghosts = [0u8; N * N];
    let debug = std::env::var("DEBUG").is_ok();
    let mut first_bad: Option<usize> = None;
    let mut f = 0usize;
    let mut was_excused = [false; N * N];
    while let Some(px) = fr.next()? {
        let (mut cells, food) = read_frame(px, 224);
        if let Some(p) = prev.as_ref() {
            for k in 0..N * N {
                cells[k].stale_art = was_excused[k] && changed(&cells[k].samples, &p[k].samples) <= 4;
            }
        }
        if f == 0 {
            for i in 0..N {
                for j in 0..N {
                    let ci = &cells[idx((i, j))];
                    if ci.paint >= 13 {
                        let mut v = Votes::default();
                        v.add(ci.dom.unwrap());
                        snakes.push(Snake { body: vec![(i, j); 3], alive: true, votes: v });
                    }
                }
            }
        } else {
            let p = prev.as_ref().unwrap();
            // eliminated this turn: the body behind the head is now grey or empty
            for s in snakes.iter_mut().filter(|s| s.alive) {
                let rest: HashSet<Cell> = s.body[1..].iter().copied().collect();
                let grey = rest.iter().filter(|&&c| cells[idx(c)].paint == 0 && cells[idx(c)].greyish).count();
                if grey as f64 > rest.len() as f64 / 2.0 {
                    if debug {
                        eprintln!("DEATH {id} turn {f} body went grey/empty at {:?}", s.body[0]);
                    }
                    s.alive = false;
                    mark_ghost(&mut ghosts, &s.body);
                }
            }
            let moved = assign(&snakes, &cells, p, &prev_food, &ghosts);
            for (sid, s) in snakes.iter_mut().enumerate().filter(|(_, s)| s.alive) {
                let Some(best) = moved[sid] else {
                    if debug {
                        eprintln!("DEATH {id} turn {f} no move from {:?}", s.body[0]);
                    }
                    s.alive = false;
                    mark_ghost(&mut ghosts, &s.body);
                    continue;
                };
                let mut body = Vec::with_capacity(s.body.len() + 1);
                body.push(best);
                body.extend_from_slice(&s.body[..s.body.len() - 1]);
                // Battlesnake: move (the tail follows), then eating stacks a copy of the tail
                if prev_food.contains(&best) {
                    body.push(*body.last().unwrap());
                }
                s.body = body;
                // a plain body segment carries the snake's own colour
                if let Some(dc) = cells[idx(s.body[1])].dom {
                    s.votes.add(dc);
                }
            }
        }
        let covered: HashSet<Cell> = snakes.iter().filter(|s| s.alive).flat_map(|s| s.body.iter().copied()).collect();
        let ends: HashSet<Cell> = snakes.iter().filter(|s| s.alive).flat_map(|s| [s.body[0], *s.body.last().unwrap()]).collect();
        for i in 0..N {
            for j in 0..N {
                let p = cells[idx((i, j))].paint;
                let cov = covered.contains(&(i, j));
                let ci = &cells[idx((i, j))];
                was_excused[idx((i, j))] = p >= 20 && !cov && excused(ghosts[idx((i, j))], ci);
                if (p >= 20 && !cov && !excused(ghosts[idx((i, j))], ci)) || (ci.blank && cov && !ends.contains(&(i, j))) {
                    errors += 1;
                    if debug && first_bad.is_none() {
                        first_bad = Some(f);
                    }
                    if debug && first_bad == Some(f) {
                        eprintln!(
                            "DEBUG {id} turn {f} cell {:?} paint {p} {} | snakes: {:?}",
                            (i, j),
                            if cov { "covered-but-empty" } else { "painted-but-uncovered" },
                            snakes.iter().map(|s| if s.alive { format!("{:?}..len{}", &s.body[..s.body.len().min(3)], s.body.len()) } else { "dead".into() }).collect::<Vec<_>>()
                        );
                    }
                }
            }
        }
        // ownership, checked at the end against each snake's final colour
        for (sid, s) in snakes.iter().enumerate().filter(|(_, s)| s.alive) {
            let (head, tail) = (s.body[0], *s.body.last().unwrap());
            let mid: HashSet<Cell> = s.body[1..s.body.len() - 1].iter().copied().filter(|&c| c != head && c != tail).collect();
            for c in mid {
                let ci = &cells[idx(c)];
                if let (Some(dc), true) = (ci.dom, ci.paint >= 20) {
                    let alive: Vec<usize> = snakes.iter().enumerate().filter(|(_, o)| o.alive).map(|(k, _)| k).collect();
                    owned.push((sid, dc, alive));
                }
            }
        }
        let mut fd: Vec<[usize; 2]> = food.iter().map(|&(x, y)| [x, N - 1 - y]).collect();
        fd.sort();
        let sn: Vec<serde_json::Value> = snakes
            .iter()
            .map(|s| {
                if s.alive {
                    serde_json::json!(s.body.iter().map(|&(x, y)| [x, N - 1 - y]).collect::<Vec<_>>())
                } else {
                    serde_json::Value::Null
                }
            })
            .collect();
        turns.push(serde_json::json!({"food": fd, "snakes": sn}));
        prev_food = food;
        prev = Some(cells);
        f += 1;
    }
    let finals: Vec<Rgb> = snakes.iter().map(|s| s.votes.top()).collect();
    errors += owned
        .iter()
        .filter(|(sid, dc, alive)| {
            let others: Vec<Rgb> = alive.iter().filter(|&&k| k != *sid).map(|&k| finals[k]).collect();
            !belongs(*dc, finals[*sid], &others)
        })
        .count();
    let colors: Vec<String> = snakes
        .iter()
        .map(|s| {
            let c = s.votes.top();
            format!("#{:02x}{:02x}{:02x}", c.0, c.1, c.2)
        })
        .collect();
    let (nt, ns) = (turns.len(), snakes.len());
    let g = serde_json::json!({"id": id, "w": N, "h": N, "colors": colors, "errors": errors, "turns": turns});
    Ok(Game { json: g.to_string(), turns: nt, snakes: ns, errors })
}

/// Where the GIFs come from: a local directory, or gs://bucket read straight into memory.
enum Source {
    Dir(PathBuf),
    Gcs { bucket: String, token: std::sync::RwLock<(String, std::time::Instant)> },
}

impl Source {
    fn gcs_token() -> String {
        let v: serde_json::Value = ureq::get(
            "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token",
        )
        .set("Metadata-Flavor", "Google")
        .call()
        .expect("metadata token")
        .into_string()
        .map(|t| serde_json::from_str(&t).unwrap())
        .unwrap();
        v["access_token"].as_str().unwrap().to_string()
    }

    fn token(&self) -> String {
        let Source::Gcs { token, .. } = self else { unreachable!() };
        {
            let t = token.read().unwrap();
            if t.1.elapsed().as_secs() < 1800 {
                return t.0.clone();
            }
        }
        let fresh = Self::gcs_token();
        *token.write().unwrap() = (fresh.clone(), std::time::Instant::now());
        fresh
    }

    fn list(&self) -> Vec<String> {
        match self {
            Source::Dir(d) => {
                let mut v: Vec<String> = std::fs::read_dir(d)
                    .unwrap()
                    .filter_map(|e| e.ok().map(|e| e.file_name().to_string_lossy().into_owned()))
                    .filter(|n| n.ends_with(".gif"))
                    .collect();
                v.sort();
                v
            }
            Source::Gcs { bucket, .. } => {
                let (mut names, mut page) = (Vec::new(), String::new());
                loop {
                    let mut url = format!(
                        "https://storage.googleapis.com/storage/v1/b/{bucket}/o?maxResults=1000&fields=items(name),nextPageToken"
                    );
                    if !page.is_empty() {
                        url += &format!("&pageToken={page}");
                    }
                    let v: serde_json::Value = ureq::get(&url)
                        .set("Authorization", &format!("Bearer {}", self.token()))
                        .call()
                        .expect("list")
                        .into_string()
                        .map(|t| serde_json::from_str(&t).unwrap())
                        .unwrap();
                    for it in v["items"].as_array().into_iter().flatten() {
                        let n = it["name"].as_str().unwrap();
                        if n.ends_with(".gif") {
                            names.push(n.to_string());
                        }
                    }
                    match v["nextPageToken"].as_str() {
                        Some(t) => page = t.to_string(),
                        None => break,
                    }
                }
                names.sort();
                names
            }
        }
    }

    fn fetch(&self, name: &str) -> Result<Vec<u8>, String> {
        match self {
            Source::Dir(d) => std::fs::read(d.join(name)).map_err(|e| e.to_string()),
            Source::Gcs { bucket, .. } => {
                let url = format!("https://storage.googleapis.com/storage/v1/b/{bucket}/o/{name}?alt=media");
                let mut last = String::new();
                for _ in 0..4 {
                    match ureq::get(&url).set("Authorization", &format!("Bearer {}", self.token())).call() {
                        Ok(r) => {
                            let mut buf = Vec::with_capacity(8 << 20);
                            std::io::Read::read_to_end(&mut r.into_reader(), &mut buf).map_err(|e| e.to_string())?;
                            return Ok(buf);
                        }
                        Err(e) => last = e.to_string(),
                    }
                }
                Err(last)
            }
        }
    }
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let (src, out, summary) = (&args[1], &args[2], &args[3]);
    let source = match src.strip_prefix("gs://") {
        Some(b) => Source::Gcs {
            bucket: b.trim_end_matches('/').to_string(),
            token: std::sync::RwLock::new((Source::gcs_token(), std::time::Instant::now())),
        },
        None => Source::Dir(PathBuf::from(src)),
    };
    // downloads overlap with decoding: more threads than cores
    let threads = std::thread::available_parallelism().map(|n| n.get()).unwrap_or(4) * 3;
    rayon::ThreadPoolBuilder::new().num_threads(threads).build_global().unwrap();
    let names = source.list();
    eprintln!("{} gifs", names.len());
    let level = std::env::var("ZLEVEL").ok().and_then(|v| v.parse().ok()).unwrap_or(3);
    let mut z = zstd::Encoder::new(BufWriter::new(File::create(out).unwrap()), level).unwrap();
    let mut sum = BufWriter::new(File::create(summary).unwrap());
    writeln!(sum, "id\tturns\tsnakes\terrors\tstatus").unwrap();
    let (mut done, mut failed, mut perfect) = (0, 0, 0);
    let t0 = std::time::Instant::now();
    for chunk in names.chunks(2048) {
        let results: Vec<(String, Result<Game, String>)> = chunk
            .par_iter()
            .map(|n| {
                let id = n.trim_end_matches(".gif").to_string();
                let r = source.fetch(n).and_then(|b| decode(b, &id));
                (id, r)
            })
            .collect();
        for (id, r) in results {
            match r {
                Ok(g) => {
                    z.write_all(g.json.as_bytes()).unwrap();
                    z.write_all(b"\n").unwrap();
                    if g.errors == 0 {
                        perfect += 1;
                    }
                    writeln!(sum, "{id}\t{}\t{}\t{}\tok", g.turns, g.snakes, g.errors).unwrap();
                }
                Err(e) => {
                    failed += 1;
                    writeln!(sum, "{id}\t0\t0\t0\t{}", e.replace('\t', " ")).unwrap();
                }
            }
            done += 1;
        }
        eprintln!(
            "{done}/{} decoded ({perfect} perfect, {failed} unreadable) {:.0}s",
            names.len(),
            t0.elapsed().as_secs_f64()
        );
    }
    z.finish().unwrap().flush().unwrap();
    sum.flush().unwrap();
}
