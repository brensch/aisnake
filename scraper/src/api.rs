//! Battlesnake's public pages and engine API: who the top snakes are, which games they
//! played, and every frame of a game.

use crate::codec::{Cell, Death, Frame, Grid, SnakeState};
use serde::Deserialize;
use std::collections::HashMap;
use std::time::Duration;

const PLAY: &str = "https://play.battlesnake.com";
const ENGINE: &str = "https://engine.battlesnake.com";
/// The engine serves at most this many frames per request.
const PAGE: usize = 100;

pub struct Api {
    agent: ureq::Agent,
    /// Pause between requests, to be a polite scraper.
    pub pause: Duration,
}

#[allow(non_snake_case)]
#[derive(Deserialize)]
pub struct GameInfo {
    pub ID: String,
    pub Status: String,
    pub Width: u32,
    pub Height: u32,
    #[serde(default)]
    pub Ruleset: serde_json::Value,
    #[serde(default)]
    pub RulesetName: String,
    #[serde(default)]
    pub Map: String,
    #[serde(default)]
    pub Source: String,
    #[serde(default)]
    pub Created: String,
}

#[allow(non_snake_case)]
#[derive(Deserialize)]
struct GameResp {
    Game: GameInfo,
}

#[allow(non_snake_case)]
#[derive(Deserialize, Clone)]
struct P {
    X: i32,
    Y: i32,
}

#[allow(non_snake_case)]
#[derive(Deserialize, Clone)]
struct DeathJ {
    Cause: String,
    Turn: u32,
    #[serde(default)]
    EliminatedBy: String,
}

#[allow(non_snake_case)]
#[derive(Deserialize, Clone)]
pub struct SnakeJ {
    pub ID: String,
    #[serde(default)]
    pub Name: String,
    #[serde(default)]
    pub Author: String,
    #[serde(default)]
    pub Color: String,
    #[serde(default)]
    pub HeadType: String,
    #[serde(default)]
    pub TailType: String,
    #[serde(default)]
    pub IsBot: bool,
    Body: Vec<P>,
    Health: i32,
    Death: Option<DeathJ>,
}

#[allow(non_snake_case)]
#[derive(Deserialize)]
struct FrameJ {
    Turn: u32,
    Snakes: Vec<SnakeJ>,
    #[serde(default)]
    Food: Option<Vec<P>>,
    #[serde(default)]
    Hazards: Option<Vec<P>>,
}

#[allow(non_snake_case)]
#[derive(Deserialize)]
struct FramesResp {
    Frames: Option<Vec<FrameJ>>,
}

/// A fetched game: its info, the snakes (in the order frames are indexed by), and frames.
pub struct Game {
    pub snakes: Vec<SnakeJ>,
    pub grid: Grid,
    pub frames: Vec<Frame>,
}

impl Api {
    pub fn new(pause: Duration) -> Self {
        let agent = ureq::AgentBuilder::new()
            .timeout(Duration::from_secs(60))
            .user_agent("bsscrape (+https://github.com/brensch/aisnake/tree/main/scraper)")
            .build();
        Api { agent, pause }
    }

    fn get(&self, url: &str) -> Result<String, String> {
        std::thread::sleep(self.pause);
        let mut last = String::new();
        for attempt in 0..3 {
            match self.agent.get(url).call() {
                Ok(r) => return r.into_string().map_err(|e| e.to_string()),
                Err(ureq::Error::Status(code, _)) if code == 404 => return Err("404".into()),
                Err(e) => last = e.to_string(),
            }
            std::thread::sleep(Duration::from_secs(2 << attempt));
        }
        Err(last)
    }

    /// The top `n` snakes on a leaderboard, best first, as their URL slugs.
    pub fn leaderboard(&self, board: &str, n: usize) -> Result<Vec<String>, String> {
        let html = self.get(&format!("{PLAY}/leaderboard/{board}"))?;
        let re = regex::Regex::new(&format!(r#"href="/leaderboard/{}/([^"/]+)/stats""#, regex::escape(board))).unwrap();
        let mut out: Vec<String> = Vec::new();
        for c in re.captures_iter(&html) {
            let slug = c[1].to_string();
            if !out.contains(&slug) {
                out.push(slug);
            }
            if out.len() == n {
                break;
            }
        }
        Ok(out)
    }

    /// The ids of a snake's recent games (its stats page lists the last 100).
    pub fn recent_games(&self, board: &str, slug: &str) -> Result<Vec<String>, String> {
        let html = self.get(&format!("{PLAY}/leaderboard/{board}/{slug}/stats"))?;
        let re = regex::Regex::new(r"/game/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})").unwrap();
        let mut out: Vec<String> = Vec::new();
        for c in re.captures_iter(&html) {
            if !out.iter().any(|x| x == &c[1]) {
                out.push(c[1].to_string());
            }
        }
        Ok(out)
    }

    pub fn info(&self, id: &str) -> Result<GameInfo, String> {
        let body = self.get(&format!("{ENGINE}/games/{id}"))?;
        let r: GameResp = serde_json::from_str(&body).map_err(|e| format!("game json: {e}"))?;
        Ok(r.Game)
    }

    /// Every frame of a finished game.
    pub fn game(&self, info: GameInfo) -> Result<Game, String> {
        let mut raw: Vec<FrameJ> = Vec::new();
        loop {
            let url = format!("{ENGINE}/games/{}/frames?offset={}&limit={PAGE}", info.ID, raw.len());
            let r: FramesResp = serde_json::from_str(&self.get(&url)?).map_err(|e| format!("frames json: {e}"))?;
            let page = r.Frames.unwrap_or_default();
            let n = page.len();
            raw.extend(page);
            if n < PAGE {
                break;
            }
        }
        if raw.is_empty() {
            return Err("no frames".into());
        }
        let grid = Grid { w: info.Width, h: info.Height };
        let snakes = raw[0].Snakes.clone();
        let index: HashMap<&str, u8> = snakes.iter().enumerate().map(|(k, s)| (s.ID.as_str(), k as u8)).collect();
        let cells = |ps: &[P]| -> Result<Vec<Cell>, String> {
            ps.iter().map(|p| grid.cell(p.X, p.Y).ok_or_else(|| format!("cell out of range: {},{}", p.X, p.Y))).collect()
        };
        let mut frames = Vec::with_capacity(raw.len());
        for (i, f) in raw.iter().enumerate() {
            if f.Turn as usize != raw[0].Turn as usize + i {
                return Err(format!("frames not contiguous at turn {}", f.Turn));
            }
            let mut snakes_out = vec![None; snakes.len()];
            for s in &f.Snakes {
                let k = *index.get(s.ID.as_str()).ok_or("snake not in first frame")? as usize;
                let death = s.Death.as_ref().map(|d| Death {
                    cause: d.Cause.clone(),
                    turn: d.Turn,
                    by: index.get(d.EliminatedBy.as_str()).copied(),
                });
                snakes_out[k] = Some(SnakeState { body: cells(&s.Body)?, health: s.Health, death });
            }
            let snakes_out: Option<Vec<SnakeState>> = snakes_out.into_iter().collect();
            let mut food = cells(f.Food.as_deref().unwrap_or(&[]))?;
            let mut hazards = cells(f.Hazards.as_deref().unwrap_or(&[]))?;
            food.sort_unstable();
            hazards.sort_unstable();
            frames.push(Frame { turn: f.Turn, snakes: snakes_out.ok_or("a snake is missing from a frame")?, food, hazards });
        }
        if frames.iter().any(|f| f.snakes.iter().any(|s| s.body.is_empty())) {
            return Err("empty snake body".into());
        }
        Ok(Game { snakes, grid, frames })
    }
}
