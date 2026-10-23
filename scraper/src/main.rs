//! bsscrape: collects every turn of Battlesnake games played by the top snakes on each
//! leaderboard, into one compact SQLite file.
//!
//!   bsscrape run    [--db F] [--top N] [--every S]   discover and fetch, forever
//!   bsscrape once   [--db F] [--top N]               one round, then exit
//!   bsscrape stats  [--db F]                         what's stored, and how big
//!   bsscrape export [--db F] <game id>               one game's turns as JSON
//!   bsscrape dump   [--db F]                         every game as JSON lines (for training)
//!   bsscrape train-dict [--db F]                     (re)train the compression dictionary now

mod api;
mod codec;
mod store;

use std::time::{Duration, Instant};
use store::{SnakeRow, Store};

const BOARDS: [&str; 4] = ["standard", "standard-duels", "royale", "constrictor"];
/// Format 1: the compact codec, zstd-compressed. Format 0: the engine's frames JSON,
/// zstd-compressed, kept for any game the codec can't reproduce exactly.
const COMPACT: i64 = 1;
const RAW: i64 = 0;
const DICT_SIZE: usize = 32 * 1024;

struct Opts {
    db: String,
    top: usize,
    every: u64,
    rest: Vec<String>,
}

fn opts() -> (String, Opts) {
    let mut args = std::env::args().skip(1);
    let cmd = args.next().unwrap_or_else(|| "help".into());
    let mut o = Opts { db: "games.sqlite".into(), top: 100, every: 1800, rest: Vec::new() };
    while let Some(a) = args.next() {
        match a.as_str() {
            "--db" => o.db = args.next().expect("--db needs a path"),
            "--top" => o.top = args.next().and_then(|v| v.parse().ok()).expect("--top needs a number"),
            "--every" => o.every = args.next().and_then(|v| v.parse().ok()).expect("--every needs seconds"),
            _ => o.rest.push(a),
        }
    }
    (cmd, o)
}

fn log(msg: &str) {
    eprintln!("{msg}");
}

/// Find the recent games of every top snake and queue the new ones.
fn discover(api: &api::Api, st: &Store, top: usize) -> usize {
    let mut new = 0;
    for board in BOARDS {
        let slugs = match api.leaderboard(board, top) {
            Ok(s) => s,
            Err(e) => {
                log(&format!("leaderboard {board}: {e}"));
                continue;
            }
        };
        for slug in slugs {
            match api.recent_games(board, &slug) {
                Ok(ids) => {
                    for id in ids {
                        if st.enqueue(&id, &format!("{board}/{slug}")).unwrap_or(false) {
                            new += 1;
                        }
                    }
                }
                Err(e) => log(&format!("stats {board}/{slug}: {e}")),
            }
        }
    }
    new
}

/// Fetch, encode and store one game.
fn fetch(api: &api::Api, st: &mut Store, id: &str) -> Result<&'static str, (String, i64, bool)> {
    let info = api.info(id).map_err(|e| (e.clone(), if e == "404" { 0 } else { 600 }, true))?;
    if info.Status != "complete" {
        // still being played: check back later, without using up an attempt
        return Err((format!("status {}", info.Status), 900, false));
    }
    let created = info.Created.parse::<i64>().map(|us| us / 1_000_000).unwrap_or(0);
    let settings = info.Ruleset.to_string();
    let meta = [info.RulesetName.clone(), settings, info.Map.clone(), info.Source.clone()];
    let size = (info.Width, info.Height);
    let game = api.game(info).map_err(|e| (e, 1800, true))?;
    let constrictor = meta[0].contains("constrictor");
    let data = codec::encode(&game.grid, constrictor, &game.frames);
    // keep the compact form only if it reproduces every frame exactly
    let (format, blob) = match codec::decode(&data) {
        Ok((_, back)) if back == game.frames => (COMPACT, data),
        _ => {
            log(&format!("{id}: compact form doesn't round-trip; storing raw"));
            (RAW, frames_json(&game.grid, &game.frames).into_bytes())
        }
    };
    let dict = st.dict().map_err(|e| (e.to_string(), 600, true))?;
    let packed = store::pack(&blob, dict.as_ref().map(|d| &d.1[..])).map_err(|e| (e, 600, true))?;
    let last = game.frames.last().unwrap();
    let rows: Vec<SnakeRow> = game
        .snakes
        .iter()
        .zip(&last.snakes)
        .map(|(s, end)| SnakeRow {
            snake_id: &s.ID,
            name: &s.Name,
            author: &s.Author,
            color: &s.Color,
            head: &s.HeadType,
            tail: &s.TailType,
            is_bot: s.IsBot,
            death_turn: end.death.as_ref().map(|d| d.turn),
            death_cause: end.death.as_ref().map(|d| d.cause.as_str()),
        })
        .collect();
    let meta: [&str; 4] = [&meta[0], &meta[1], &meta[2], &meta[3]];
    st.save(id, created, meta, size, last.turn, &rows, format, dict.map(|d| d.0), &packed).map_err(|e| (e, 600, true))?;
    Ok(if format == COMPACT { "compact" } else { "raw" })
}

fn drain(api: &api::Api, st: &mut Store, budget: Duration) -> (usize, usize) {
    let start = Instant::now();
    let (mut done, mut failed) = (0, 0);
    while start.elapsed() < budget {
        let ids = st.due(50).unwrap_or_default();
        if ids.is_empty() {
            break;
        }
        for id in ids {
            match fetch(api, st, &id) {
                Ok(_) => done += 1,
                Err((e, wait, counts)) => {
                    if counts {
                        failed += 1;
                        log(&format!("{id}: {e}"));
                    }
                    let _ = st.retry(&id, &e, wait, counts, 5);
                }
            }
            if start.elapsed() >= budget {
                break;
            }
        }
    }
    (done, failed)
}

fn frames_json(g: &codec::Grid, frames: &[codec::Frame]) -> String {
    let xy = |c: codec::Cell| {
        let (x, y) = g.xy(c);
        serde_json::json!([x, y])
    };
    let out: Vec<serde_json::Value> = frames
        .iter()
        .map(|f| {
            serde_json::json!({
                "turn": f.turn,
                "food": f.food.iter().map(|&c| xy(c)).collect::<Vec<_>>(),
                "hazards": f.hazards.iter().map(|&c| xy(c)).collect::<Vec<_>>(),
                "snakes": f.snakes.iter().map(|s| serde_json::json!({
                    "body": s.body.iter().map(|&c| xy(c)).collect::<Vec<_>>(),
                    "health": s.health,
                    "death": s.death.as_ref().map(|d| serde_json::json!({"cause": d.cause, "turn": d.turn, "by": d.by})),
                })).collect::<Vec<_>>(),
            })
        })
        .collect();
    serde_json::Value::Array(out).to_string()
}

/// One stored game as JSON: its metadata, snakes and every turn.
fn game_json(st: &Store, id: &str) -> Result<String, String> {
    let (format, blob) = st.game_data(id)?.ok_or("no such game")?;
    let turns = if format == COMPACT {
        let (g, frames) = codec::decode(&blob)?;
        frames_json(&g, &frames)
    } else {
        String::from_utf8(blob).map_err(|e| e.to_string())?
    };
    let b = store::id_bytes(id)?;
    let (created, ruleset, settings, map, w, h): (i64, i64, i64, i64, u32, u32) = st
        .db
        .query_row("SELECT created, ruleset, settings, map, width, height FROM games WHERE id=?1", [&b], |r| {
            Ok((r.get(0)?, r.get(1)?, r.get(2)?, r.get(3)?, r.get(4)?, r.get(5)?))
        })
        .map_err(|e| e.to_string())?;
    let mut q = st
        .db
        .prepare("SELECT name, author, color, death_turn, death_cause FROM game_snakes WHERE game=?1 ORDER BY idx")
        .map_err(|e| e.to_string())?;
    let snakes: Vec<serde_json::Value> = q
        .query_map([&b], |r| {
            Ok((r.get::<_, i64>(0)?, r.get::<_, i64>(1)?, r.get::<_, i64>(2)?, r.get::<_, Option<u32>>(3)?, r.get::<_, Option<i64>>(4)?))
        })
        .map_err(|e| e.to_string())?
        .filter_map(Result::ok)
        .map(|(n, a, c, dt, dc)| {
            serde_json::json!({
                "name": st.get_string(n).unwrap_or_default(),
                "author": st.get_string(a).unwrap_or_default(),
                "color": st.get_string(c).unwrap_or_default(),
                "death_turn": dt,
                "death_cause": dc.and_then(|x| st.get_string(x).ok()),
            })
        })
        .collect();
    Ok(format!(
        r#"{{"id":{},"created":{created},"ruleset":{},"settings":{},"map":{},"width":{w},"height":{h},"snakes":{},"turns":{turns}}}"#,
        serde_json::json!(id),
        serde_json::json!(st.get_string(ruleset).unwrap_or_default()),
        st.get_string(settings).unwrap_or_else(|_| "null".into()),
        serde_json::json!(st.get_string(map).unwrap_or_default()),
        serde_json::Value::Array(snakes)
    ))
}

fn stats(st: &Store) -> rusqlite::Result<()> {
    let q = |sql: &str| st.db.query_row(sql, [], |r| r.get::<_, i64>(0));
    let games = q("SELECT count(*) FROM games")?;
    let raw = q("SELECT count(*) FROM games WHERE format = 0")?;
    let turns = q("SELECT coalesce(sum(turns), 0) FROM games")?;
    let bytes = q("SELECT coalesce(sum(length(data)), 0) FROM games")?;
    let queued = q("SELECT count(*) FROM queue WHERE status = 0")?;
    let gave_up = q("SELECT count(*) FROM queue WHERE status = 2")?;
    let file = q("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()")?;
    println!("games stored   {games} ({raw} raw), {turns} turns");
    println!("game data      {} bytes, {:.0} bytes/game, {:.2} bytes/turn", bytes, bytes as f64 / games.max(1) as f64, bytes as f64 / turns.max(1) as f64);
    println!("database file  {:.1} MB", file as f64 / 1e6);
    println!("queue          {queued} to fetch, {gave_up} given up");
    let mut by = st.db.prepare(
        "SELECT s.s, count(*) FROM games g JOIN strings s ON s.id = g.ruleset GROUP BY 1 ORDER BY 2 DESC",
    )?;
    for row in by.query_map([], |r| Ok((r.get::<_, String>(0)?, r.get::<_, i64>(1)?)))? {
        let (k, n) = row?;
        println!("  {k:<14} {n}");
    }
    Ok(())
}

fn main() {
    let (cmd, o) = opts();
    let mut st = Store::open(&o.db).expect("open database");
    let api = api::Api::new(Duration::from_millis(250));
    match cmd.as_str() {
        "run" | "once" => loop {
            let round = Instant::now();
            let new = discover(&api, &st, o.top);
            let budget = Duration::from_secs(o.every).saturating_sub(round.elapsed());
            let (done, failed) = drain(&api, &mut st, if cmd == "once" { Duration::from_secs(86_400) } else { budget });
            let queued = st.due(1_000_000).map(|v| v.len()).unwrap_or(0);
            log(&format!(
                "round: {new} new games found, {done} stored, {failed} failed, {queued} due ({:.0}s)",
                round.elapsed().as_secs_f64()
            ));
            // once there is enough to learn from, train a compression dictionary (and again
            // at 50k games, when the data is far more representative)
            let games: i64 = st.db.query_row("SELECT count(*) FROM games", [], |r| r.get(0)).unwrap_or(0);
            let dicts: i64 = st.db.query_row("SELECT count(*) FROM dicts", [], |r| r.get(0)).unwrap_or(0);
            if (dicts == 0 && games >= 2_000) || (dicts == 1 && games >= 50_000) {
                match st.train_dict(DICT_SIZE) {
                    Ok((n, bytes)) => log(&format!("trained a dictionary; recompressed {n} games to {bytes} bytes")),
                    Err(e) => log(&format!("dictionary training failed: {e}")),
                }
            }
            if cmd == "once" {
                break;
            }
            std::thread::sleep(Duration::from_secs(o.every).saturating_sub(round.elapsed()));
        },
        "stats" => stats(&st).expect("stats"),
        "train-dict" => match st.train_dict(DICT_SIZE) {
            Ok((n, bytes)) => println!("recompressed {n} games to {bytes} bytes"),
            Err(e) => eprintln!("error: {e}"),
        },
        "export" => match o.rest.first() {
            Some(id) => println!("{}", game_json(&st, id).unwrap_or_else(|e| format!("error: {e}"))),
            None => eprintln!("usage: bsscrape export <game id>"),
        },
        "dump" => {
            let ids: Vec<Vec<u8>> = {
                let mut q = st.db.prepare("SELECT id FROM games ORDER BY created").expect("query");
                q.query_map([], |r| r.get(0)).expect("query").filter_map(Result::ok).collect()
            };
            for b in ids {
                match game_json(&st, &store::id_string(&b)) {
                    Ok(j) => println!("{j}"),
                    Err(e) => eprintln!("{}: {e}", store::id_string(&b)),
                }
            }
        }
        _ => eprintln!("{}", include_str!("main.rs").lines().take(10).map(|l| l.trim_start_matches("//!")).collect::<Vec<_>>().join("\n")),
    }
}
