//! The compact game format: the first frame in full, then per turn only what the rules
//! don't already imply.
//!
//! Per turn, each snake that was alive costs 2 bits (the direction it moved). The new body
//! is predicted from that (new head, tail follows, tail stacks when it ate or in
//! constrictor) and the health too (100 when it ate, otherwise one less). Anything else is
//! written out as an exception: food and hazard changes, eliminations, and any body or
//! health the prediction got wrong (hazard damage, for example). Dead snakes are predicted
//! to stay as they were. `decode(encode(frames)) == frames` is checked before a game is
//! stored, so the format is lossless for everything it keeps.

use std::collections::BTreeMap;

/// A board cell as an index into a (w+2)x(h+2) grid, so a head one step off the board
/// (a wall collision) still fits.
pub type Cell = u32;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Death {
    pub cause: String,
    pub turn: u32,
    /// Index of the snake that caused it.
    pub by: Option<u8>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SnakeState {
    pub body: Vec<Cell>,
    pub health: i32,
    pub death: Option<Death>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Frame {
    pub turn: u32,
    pub snakes: Vec<SnakeState>,
    /// Sorted. Hazards can repeat (stacked hazards), so these are multisets.
    pub food: Vec<Cell>,
    pub hazards: Vec<Cell>,
}

pub struct Grid {
    pub w: u32,
    pub h: u32,
}

impl Grid {
    pub fn cell(&self, x: i32, y: i32) -> Option<Cell> {
        if x < -1 || y < -1 || x > self.w as i32 || y > self.h as i32 {
            return None;
        }
        Some((y + 1) as u32 * (self.w + 2) + (x + 1) as u32)
    }
    pub fn xy(&self, c: Cell) -> (i32, i32) {
        ((c % (self.w + 2)) as i32 - 1, (c / (self.w + 2)) as i32 - 1)
    }
    fn step(&self, from: Cell, dir: u8) -> Cell {
        let (x, y) = self.xy(from);
        let (x, y) = match dir {
            0 => (x, y + 1),
            1 => (x, y - 1),
            2 => (x - 1, y),
            _ => (x + 1, y),
        };
        self.cell(x, y).unwrap_or(u32::MAX)
    }
    fn dir(&self, from: Cell, to: Cell) -> Option<u8> {
        let ((ax, ay), (bx, by)) = (self.xy(from), self.xy(to));
        match (bx - ax, by - ay) {
            (0, 1) => Some(0),
            (0, -1) => Some(1),
            (-1, 0) => Some(2),
            (1, 0) => Some(3),
            _ => None,
        }
    }
}

// ---- byte writing ----

struct W(Vec<u8>);

impl W {
    fn u(&mut self, mut v: u64) {
        loop {
            let b = (v & 0x7f) as u8;
            v >>= 7;
            if v == 0 {
                self.0.push(b);
                return;
            }
            self.0.push(b | 0x80);
        }
    }
    fn i(&mut self, v: i64) {
        self.u(((v << 1) ^ (v >> 63)) as u64);
    }
    fn cells(&mut self, cs: &[Cell]) {
        self.u(cs.len() as u64);
        for &c in cs {
            self.u(c as u64);
        }
    }
    fn s(&mut self, s: &str) {
        self.u(s.len() as u64);
        self.0.extend_from_slice(s.as_bytes());
    }
}

struct R<'a>(&'a [u8], usize);

impl R<'_> {
    fn u(&mut self) -> Result<u64, String> {
        let (mut v, mut shift) = (0u64, 0);
        loop {
            let b = *self.0.get(self.1).ok_or("truncated")?;
            self.1 += 1;
            v |= ((b & 0x7f) as u64) << shift;
            if b & 0x80 == 0 {
                return Ok(v);
            }
            shift += 7;
            if shift > 63 {
                return Err("bad varint".into());
            }
        }
    }
    fn i(&mut self) -> Result<i64, String> {
        let v = self.u()?;
        Ok((v >> 1) as i64 ^ -((v & 1) as i64))
    }
    fn byte(&mut self) -> Result<u8, String> {
        let b = *self.0.get(self.1).ok_or("truncated")?;
        self.1 += 1;
        Ok(b)
    }
    fn cells(&mut self) -> Result<Vec<Cell>, String> {
        let n = self.u()? as usize;
        (0..n).map(|_| self.u().map(|c| c as Cell)).collect()
    }
    fn s(&mut self) -> Result<String, String> {
        let n = self.u()? as usize;
        let b = self.0.get(self.1..self.1 + n).ok_or("truncated")?;
        self.1 += n;
        String::from_utf8(b.to_vec()).map_err(|e| e.to_string())
    }
}

/// (added, removed) between two sorted multisets.
fn diff(prev: &[Cell], cur: &[Cell]) -> (Vec<Cell>, Vec<Cell>) {
    let mut count: BTreeMap<Cell, i32> = BTreeMap::new();
    for &c in cur {
        *count.entry(c).or_default() += 1;
    }
    for &c in prev {
        *count.entry(c).or_default() -= 1;
    }
    let (mut add, mut rem) = (Vec::new(), Vec::new());
    for (c, n) in count {
        for _ in 0..n.max(0) {
            add.push(c);
        }
        for _ in 0..(-n).max(0) {
            rem.push(c);
        }
    }
    (add, rem)
}

fn apply(prev: &[Cell], add: &[Cell], rem: &[Cell]) -> Vec<Cell> {
    let mut out = prev.to_vec();
    for c in rem {
        if let Some(i) = out.iter().position(|x| x == c) {
            out.remove(i);
        }
    }
    out.extend_from_slice(add);
    out.sort_unstable();
    out
}

/// What the rules predict for a snake next turn, given the direction it moved.
fn predict(g: &Grid, prev: &SnakeState, food: &[Cell], dir: u8, constrictor: bool) -> (Vec<Cell>, i32) {
    let head = g.step(prev.body[0], dir);
    let mut body = Vec::with_capacity(prev.body.len() + 1);
    body.push(head);
    body.extend_from_slice(&prev.body[..prev.body.len() - 1]);
    let ate = food.binary_search(&head).is_ok();
    if ate || constrictor {
        body.push(*body.last().unwrap());
    }
    (body, if ate { 100 } else { prev.health - 1 })
}

const F_FOOD: u8 = 1;
const F_HAZARD: u8 = 2;
const F_BODY: u8 = 4;
const F_HEALTH: u8 = 8;
const F_DEATH: u8 = 16;

pub fn encode(g: &Grid, constrictor: bool, frames: &[Frame]) -> Vec<u8> {
    let mut w = W(Vec::new());
    w.0.push(1); // format version
    w.u(g.w as u64);
    w.u(g.h as u64);
    w.0.push(constrictor as u8);
    w.u(frames.len() as u64);
    let f0 = &frames[0];
    w.u(f0.turn as u64);
    w.u(f0.snakes.len() as u64);
    for s in &f0.snakes {
        w.cells(&s.body);
        w.i(s.health as i64);
        write_death(&mut w, &s.death);
    }
    w.cells(&f0.food);
    w.cells(&f0.hazards);

    for pair in frames.windows(2) {
        let (p, c) = (&pair[0], &pair[1]);
        w.u((c.turn - p.turn) as u64);
        let mut bits: Vec<u8> = Vec::new();
        let (mut bodies, mut healths, mut deaths) = (Vec::new(), Vec::new(), Vec::new());
        for (k, (ps, cs)) in p.snakes.iter().zip(&c.snakes).enumerate() {
            if ps.death.is_none() {
                // moving snake: direction from the new head, then check the prediction
                let dir = g.dir(ps.body[0], cs.body[0]).unwrap_or(0);
                bits.push(dir);
                let (body, health) = predict(g, ps, &p.food, dir, constrictor);
                if body != cs.body {
                    bodies.push(k);
                }
                if health != cs.health {
                    healths.push(k);
                }
            } else {
                if ps.body != cs.body {
                    bodies.push(k);
                }
                if ps.health != cs.health {
                    healths.push(k);
                }
            }
            if ps.death != cs.death {
                deaths.push(k);
            }
        }
        let (fa, fr) = diff(&p.food, &c.food);
        let (ha, hr) = diff(&p.hazards, &c.hazards);
        let mut flags = 0;
        if !fa.is_empty() || !fr.is_empty() {
            flags |= F_FOOD;
        }
        if !ha.is_empty() || !hr.is_empty() {
            flags |= F_HAZARD;
        }
        if !bodies.is_empty() {
            flags |= F_BODY;
        }
        if !healths.is_empty() {
            flags |= F_HEALTH;
        }
        if !deaths.is_empty() {
            flags |= F_DEATH;
        }
        // 2 bits per moving snake, then the flags
        for chunk in bits.chunks(4) {
            let mut b = 0u8;
            for (i, d) in chunk.iter().enumerate() {
                b |= d << (2 * i);
            }
            w.0.push(b);
        }
        w.0.push(flags);
        if flags & F_FOOD != 0 {
            w.cells(&fa);
            w.cells(&fr);
        }
        if flags & F_HAZARD != 0 {
            w.cells(&ha);
            w.cells(&hr);
        }
        if flags & F_BODY != 0 {
            w.u(bodies.len() as u64);
            for &k in &bodies {
                w.u(k as u64);
                w.cells(&c.snakes[k].body);
            }
        }
        if flags & F_HEALTH != 0 {
            w.u(healths.len() as u64);
            for &k in &healths {
                w.u(k as u64);
                w.i(c.snakes[k].health as i64);
            }
        }
        if flags & F_DEATH != 0 {
            w.u(deaths.len() as u64);
            for &k in &deaths {
                w.u(k as u64);
                write_death(&mut w, &c.snakes[k].death);
            }
        }
    }
    w.0
}

fn write_death(w: &mut W, d: &Option<Death>) {
    match d {
        None => w.0.push(0),
        Some(d) => {
            w.0.push(1);
            w.s(&d.cause);
            w.u(d.turn as u64);
            w.u(d.by.map_or(0, |b| b as u64 + 1));
        }
    }
}

fn read_death(r: &mut R) -> Result<Option<Death>, String> {
    if r.byte()? == 0 {
        return Ok(None);
    }
    let cause = r.s()?;
    let turn = r.u()? as u32;
    let by = match r.u()? {
        0 => None,
        n => Some((n - 1) as u8),
    };
    Ok(Some(Death { cause, turn, by }))
}

pub fn decode(data: &[u8]) -> Result<(Grid, Vec<Frame>), String> {
    let mut r = R(data, 0);
    if r.byte()? != 1 {
        return Err("unknown format version".into());
    }
    let g = Grid { w: r.u()? as u32, h: r.u()? as u32 };
    let constrictor = r.byte()? != 0;
    let n = r.u()? as usize;
    let turn = r.u()? as u32;
    let ns = r.u()? as usize;
    let mut snakes = Vec::with_capacity(ns);
    for _ in 0..ns {
        let body = r.cells()?;
        let health = r.i()? as i32;
        let death = read_death(&mut r)?;
        snakes.push(SnakeState { body, health, death });
    }
    let food = r.cells()?;
    let hazards = r.cells()?;
    let mut frames = vec![Frame { turn, snakes, food, hazards }];
    while frames.len() < n {
        let p = frames.last().unwrap();
        let turn = p.turn + r.u()? as u32;
        let moving: Vec<usize> = (0..p.snakes.len()).filter(|&k| p.snakes[k].death.is_none()).collect();
        let mut dirs = Vec::with_capacity(moving.len());
        for chunk in moving.chunks(4) {
            let b = r.byte()?;
            for i in 0..chunk.len() {
                dirs.push((b >> (2 * i)) & 3);
            }
        }
        let flags = r.byte()?;
        let mut snakes = p.snakes.clone();
        for (&k, &dir) in moving.iter().zip(&dirs) {
            let (body, health) = predict(&g, &p.snakes[k], &p.food, dir, constrictor);
            snakes[k].body = body;
            snakes[k].health = health;
        }
        let mut food = p.food.clone();
        let mut hazards = p.hazards.clone();
        if flags & F_FOOD != 0 {
            let (a, rm) = (r.cells()?, r.cells()?);
            food = apply(&food, &a, &rm);
        }
        if flags & F_HAZARD != 0 {
            let (a, rm) = (r.cells()?, r.cells()?);
            hazards = apply(&hazards, &a, &rm);
        }
        if flags & F_BODY != 0 {
            for _ in 0..r.u()? {
                let k = r.u()? as usize;
                snakes.get_mut(k).ok_or("bad snake")?.body = r.cells()?;
            }
        }
        if flags & F_HEALTH != 0 {
            for _ in 0..r.u()? {
                let k = r.u()? as usize;
                snakes.get_mut(k).ok_or("bad snake")?.health = r.i()? as i32;
            }
        }
        if flags & F_DEATH != 0 {
            for _ in 0..r.u()? {
                let k = r.u()? as usize;
                snakes.get_mut(k).ok_or("bad snake")?.death = read_death(&mut r)?;
            }
        }
        frames.push(Frame { turn, snakes, food, hazards });
    }
    if r.1 != data.len() {
        return Err("trailing bytes".into());
    }
    Ok((g, frames))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn s(body: Vec<Cell>, health: i32) -> SnakeState {
        SnakeState { body, health, death: None }
    }

    #[test]
    fn round_trips_moves_growth_food_and_death() {
        let g = Grid { w: 11, h: 11 };
        let c = |x, y| g.cell(x, y).unwrap();
        let f0 = Frame {
            turn: 0,
            snakes: vec![s(vec![c(1, 1); 3], 100), s(vec![c(9, 9); 3], 100)],
            food: vec![c(1, 2), c(5, 5)],
            hazards: vec![],
        };
        // snake 0 moves up onto food and grows; snake 1 moves left
        let f1 = Frame {
            turn: 1,
            snakes: vec![s(vec![c(1, 2), c(1, 1), c(1, 1), c(1, 1)], 100), s(vec![c(8, 9), c(9, 9), c(9, 9)], 99)],
            food: vec![c(5, 5), c(7, 7)],
            hazards: vec![],
        };
        // snake 1 runs off the board and dies
        let mut dead = s(vec![c(8, 10), c(8, 9), c(9, 9)], 98);
        dead.death = Some(Death { cause: "wall-collision".into(), turn: 2, by: None });
        let f2 = Frame {
            turn: 2,
            snakes: vec![s(vec![c(2, 2), c(1, 2), c(1, 1), c(1, 1)], 99), dead.clone()],
            food: vec![c(5, 5), c(7, 7)],
            hazards: vec![c(0, 0), c(0, 0)],
        };
        let f3 = Frame { turn: 3, snakes: vec![s(vec![c(3, 2), c(2, 2), c(1, 2), c(1, 1)], 85), dead], ..f2.clone() };
        let frames = vec![f0, f1, f2, f3];
        let data = encode(&g, false, &frames);
        let (_, back) = decode(&data).unwrap();
        assert_eq!(back, frames);
    }

    #[test]
    fn off_board_cells_fit() {
        let g = Grid { w: 11, h: 11 };
        assert_eq!(g.xy(g.cell(-1, 11).unwrap()), (-1, 11));
        assert!(g.cell(-2, 0).is_none());
    }
}
