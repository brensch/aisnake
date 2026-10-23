//! SQLite: the queue of games still to fetch, and the games themselves.
//!
//! `games.data` holds the compact format (codec.rs), zstd-compressed (level 19, no frame
//! checksum), with a dictionary trained on the stored games once there are enough of them
//! (`games.dict`, `dicts`). Repeated strings (snake names, authors, rulesets...) are stored once in
//! `strings`. A game id is in `games` once fetched, or in `queue` while waiting, so nothing
//! is fetched twice.

use rusqlite::{Connection, OptionalExtension, params};

pub struct Store {
    pub db: Connection,
}

pub fn id_bytes(id: &str) -> Result<Vec<u8>, String> {
    let hex: String = id.chars().filter(|c| *c != '-').collect();
    if hex.len() != 32 {
        return Err(format!("not a game id: {id}"));
    }
    (0..16).map(|i| u8::from_str_radix(&hex[2 * i..2 * i + 2], 16).map_err(|e| e.to_string())).collect()
}

pub fn id_string(b: &[u8]) -> String {
    let h: String = b.iter().map(|x| format!("{x:02x}")).collect();
    format!("{}-{}-{}-{}-{}", &h[0..8], &h[8..12], &h[12..16], &h[16..20], &h[20..32])
}

pub fn now() -> i64 {
    std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_secs() as i64
}

pub struct SnakeRow<'a> {
    pub snake_id: &'a str,
    pub name: &'a str,
    pub author: &'a str,
    pub color: &'a str,
    pub head: &'a str,
    pub tail: &'a str,
    pub is_bot: bool,
    pub death_turn: Option<u32>,
    pub death_cause: Option<&'a str>,
}

impl Store {
    pub fn open(path: &str) -> rusqlite::Result<Self> {
        let db = Connection::open(path)?;
        db.execute_batch(
            "PRAGMA journal_mode=WAL;
             PRAGMA synchronous=NORMAL;
             CREATE TABLE IF NOT EXISTS strings(id INTEGER PRIMARY KEY, s TEXT NOT NULL UNIQUE);
             -- games found but not yet stored; status 0 = to fetch, 2 = gave up
             CREATE TABLE IF NOT EXISTS queue(
               id BLOB PRIMARY KEY, found_at INTEGER NOT NULL, found_via INTEGER,
               status INTEGER NOT NULL DEFAULT 0, attempts INTEGER NOT NULL DEFAULT 0,
               next_try INTEGER NOT NULL DEFAULT 0, last_error TEXT) WITHOUT ROWID;
             CREATE TABLE IF NOT EXISTS games(
               id BLOB PRIMARY KEY, created INTEGER, fetched INTEGER NOT NULL,
               ruleset INTEGER, settings INTEGER, map INTEGER, source INTEGER,
               width INTEGER, height INTEGER, turns INTEGER, n_snakes INTEGER,
               format INTEGER NOT NULL, data BLOB NOT NULL) WITHOUT ROWID;
             CREATE TABLE IF NOT EXISTS game_snakes(
               game BLOB NOT NULL, idx INTEGER NOT NULL, snake_id INTEGER, name INTEGER,
               author INTEGER, color INTEGER, head INTEGER, tail INTEGER, is_bot INTEGER,
               death_turn INTEGER, death_cause INTEGER,
               PRIMARY KEY(game, idx)) WITHOUT ROWID;
             CREATE INDEX IF NOT EXISTS game_snakes_author ON game_snakes(author);
             -- zstd dictionaries trained on the stored games (games.dict refers to one)
             CREATE TABLE IF NOT EXISTS dicts(id INTEGER PRIMARY KEY, created INTEGER NOT NULL, data BLOB NOT NULL);
             CREATE INDEX IF NOT EXISTS queue_next ON queue(status, next_try);",
        )?;
        // games.dict: added after the first version
        let has_dict: bool = db.prepare("SELECT 1 FROM pragma_table_info('games') WHERE name='dict'")?.exists([])?;
        if !has_dict {
            db.execute("ALTER TABLE games ADD COLUMN dict INTEGER", [])?;
        }
        Ok(Store { db })
    }

    /// The newest dictionary, if one has been trained.
    pub fn dict(&self) -> rusqlite::Result<Option<(i64, Vec<u8>)>> {
        self.db.query_row("SELECT id, data FROM dicts ORDER BY id DESC LIMIT 1", [], |r| Ok((r.get(0)?, r.get(1)?))).optional()
    }

    pub fn dict_by_id(&self, id: i64) -> rusqlite::Result<Vec<u8>> {
        self.db.query_row("SELECT data FROM dicts WHERE id=?1", [id], |r| r.get(0))
    }

    pub fn string(&self, s: &str) -> rusqlite::Result<i64> {
        if let Some(id) = self.db.query_row("SELECT id FROM strings WHERE s=?1", [s], |r| r.get(0)).optional()? {
            return Ok(id);
        }
        self.db.execute("INSERT INTO strings(s) VALUES (?1)", [s])?;
        Ok(self.db.last_insert_rowid())
    }

    pub fn get_string(&self, id: i64) -> rusqlite::Result<String> {
        self.db.query_row("SELECT s FROM strings WHERE id=?1", [id], |r| r.get(0))
    }

    /// Queue a game unless it is already queued or stored. True if it was new.
    pub fn enqueue(&self, id: &str, via: &str) -> Result<bool, String> {
        let b = id_bytes(id)?;
        let via = self.string(via).map_err(|e| e.to_string())?;
        let n = self
            .db
            .execute(
                "INSERT OR IGNORE INTO queue(id, found_at, found_via)
                 SELECT ?1, ?2, ?3 WHERE NOT EXISTS (SELECT 1 FROM games WHERE id=?1)",
                params![b, now(), via],
            )
            .map_err(|e| e.to_string())?;
        Ok(n > 0)
    }

    /// Up to `n` queued games that are due.
    pub fn due(&self, n: usize) -> rusqlite::Result<Vec<String>> {
        let mut st = self.db.prepare("SELECT id FROM queue WHERE status=0 AND next_try<=?1 ORDER BY found_at LIMIT ?2")?;
        let rows = st.query_map(params![now(), n as i64], |r| r.get::<_, Vec<u8>>(0))?;
        rows.map(|r| r.map(|b| id_string(&b))).collect()
    }

    /// Try again later: soon for a game still in progress, backing off for errors, and
    /// giving up after `max` attempts.
    pub fn retry(&self, id: &str, error: &str, wait: i64, counts: bool, max: i64) -> Result<(), String> {
        let b = id_bytes(id)?;
        self.db
            .execute(
                "UPDATE queue SET attempts = attempts + ?2, next_try = ?3, last_error = ?4,
                 status = CASE WHEN attempts + ?2 >= ?5 THEN 2 ELSE 0 END WHERE id=?1",
                params![b, counts as i64, now() + wait, error, max],
            )
            .map(|_| ())
            .map_err(|e| e.to_string())
    }

    pub fn save(
        &mut self,
        id: &str,
        created: i64,
        meta: [&str; 4], // ruleset, settings json, map, source
        size: (u32, u32),
        turns: u32,
        snakes: &[SnakeRow],
        format: i64,
        dict: Option<i64>,
        data: &[u8],
    ) -> Result<(), String> {
        let b = id_bytes(id)?;
        let ids: Vec<i64> = meta.iter().map(|s| self.string(s)).collect::<rusqlite::Result<_>>().map_err(|e| e.to_string())?;
        let mut rows = Vec::new();
        for s in snakes {
            let get = |v: &str| self.string(v);
            rows.push((
                get(s.snake_id).map_err(|e| e.to_string())?,
                get(s.name).map_err(|e| e.to_string())?,
                get(s.author).map_err(|e| e.to_string())?,
                get(s.color).map_err(|e| e.to_string())?,
                get(s.head).map_err(|e| e.to_string())?,
                get(s.tail).map_err(|e| e.to_string())?,
                s.is_bot,
                s.death_turn,
                s.death_cause.map(get).transpose().map_err(|e| e.to_string())?,
            ));
        }
        let tx = self.db.transaction().map_err(|e| e.to_string())?;
        tx.execute(
            "INSERT OR REPLACE INTO games(id, created, fetched, ruleset, settings, map, source, width, height,
               turns, n_snakes, format, data, dict) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14)",
            params![b, created, now(), ids[0], ids[1], ids[2], ids[3], size.0, size.1, turns, snakes.len(), format, data, dict],
        )
        .map_err(|e| e.to_string())?;
        for (k, r) in rows.iter().enumerate() {
            tx.execute(
                "INSERT OR REPLACE INTO game_snakes VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11)",
                params![b, k, r.0, r.1, r.2, r.3, r.4, r.5, r.6, r.7, r.8],
            )
            .map_err(|e| e.to_string())?;
        }
        tx.execute("DELETE FROM queue WHERE id=?1", [&b]).map_err(|e| e.to_string())?;
        tx.commit().map_err(|e| e.to_string())
    }

    /// A stored game's format and its decompressed bytes.
    pub fn game_data(&self, id: &str) -> Result<Option<(i64, Vec<u8>)>, String> {
        let b = id_bytes(id)?;
        let row: Option<(i64, Vec<u8>, Option<i64>)> = self
            .db
            .query_row("SELECT format, data, dict FROM games WHERE id=?1", [b], |r| Ok((r.get(0)?, r.get(1)?, r.get(2)?)))
            .optional()
            .map_err(|e| e.to_string())?;
        let Some((format, packed, dict)) = row else { return Ok(None) };
        let dict = dict.map(|d| self.dict_by_id(d)).transpose().map_err(|e| e.to_string())?;
        Ok(Some((format, unpack(&packed, dict.as_deref())?)))
    }
}

pub fn pack(data: &[u8], dict: Option<&[u8]>) -> Result<Vec<u8>, String> {
    let mut c = match dict {
        Some(d) => zstd::bulk::Compressor::with_dictionary(19, d),
        None => zstd::bulk::Compressor::new(19),
    }
    .map_err(|e| e.to_string())?;
    // no frame checksum or size header: every byte counts at ~500 bytes a game
    c.include_checksum(false).map_err(|e| e.to_string())?;
    c.include_contentsize(true).map_err(|e| e.to_string())?;
    c.include_dictid(false).map_err(|e| e.to_string())?;
    c.compress(data).map_err(|e| e.to_string())
}

pub fn unpack(packed: &[u8], dict: Option<&[u8]>) -> Result<Vec<u8>, String> {
    let mut d = match dict {
        Some(d) => zstd::bulk::Decompressor::with_dictionary(d),
        None => zstd::bulk::Decompressor::new(),
    }
    .map_err(|e| e.to_string())?;
    d.decompress(packed, 64 << 20).map_err(|e| e.to_string())
}

impl Store {
    /// Train a dictionary on the stored compact games and recompress all of them with it.
    pub fn train_dict(&mut self, size: usize) -> Result<(usize, usize), String> {
        let rows: Vec<(Vec<u8>, i64, Vec<u8>, Option<i64>)> = {
            let mut q = self.db.prepare("SELECT id, format, data, dict FROM games").map_err(|e| e.to_string())?;
            q.query_map([], |r| Ok((r.get(0)?, r.get(1)?, r.get(2)?, r.get(3)?)))
                .map_err(|e| e.to_string())?
                .filter_map(Result::ok)
                .collect()
        };
        let mut plain = Vec::with_capacity(rows.len());
        for (id, format, packed, dict) in rows {
            let d = dict.map(|d| self.dict_by_id(d)).transpose().map_err(|e| e.to_string())?;
            plain.push((id, format, unpack(&packed, d.as_deref())?));
        }
        let samples: Vec<&[u8]> = plain.iter().filter(|p| p.1 == 1).map(|p| &p.2[..]).collect();
        let dict = zstd::dict::from_samples(&samples, size).map_err(|e| e.to_string())?;
        let tx = self.db.transaction().map_err(|e| e.to_string())?;
        tx.execute("INSERT INTO dicts(created, data) VALUES (?1, ?2)", params![now(), dict]).map_err(|e| e.to_string())?;
        let dict_id = tx.last_insert_rowid();
        let (mut before, mut after) = (0, 0);
        for (id, _, data) in &plain {
            let packed = pack(data, Some(&dict))?;
            after += packed.len();
            before += 1;
            tx.execute("UPDATE games SET data=?2, dict=?3 WHERE id=?1", params![id, packed, dict_id]).map_err(|e| e.to_string())?;
        }
        tx.commit().map_err(|e| e.to_string())?;
        self.db.execute_batch("VACUUM").map_err(|e| e.to_string())?;
        Ok((before, after))
    }
}
