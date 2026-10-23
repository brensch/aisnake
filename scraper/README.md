# scraper (bsscrape)

Collects every turn of the Battlesnake games played by the top snakes, for training.
Runs on a free-tier e2-micro (`bsscrape`, project `snake2-338906`, chase billing) and keeps
everything in one SQLite file on its 30 GB disk: `/var/lib/bsscrape/games.sqlite`.

## What it scrapes

Every 30 minutes it reads the four leaderboards (standard, standard-duels, royale,
constrictor), takes the top 100 snakes on each, and reads each one's stats page (its last
100 games). Any game it hasn't seen is queued, then fetched from the engine
(`engine.battlesnake.com/games/<id>/frames`) one request at a time with a short pause, once
the game is finished. Stored games are never fetched again; failures back off and give up
after 5 attempts.

## How it stores them

About 1.4 bytes a turn. Each game is the first frame in full, then per turn 2 bits per live
snake for the direction it moved; bodies and health follow from the rules (tail follows,
eating grows the tail and resets health to 100). Only what the rules don't predict is written:
food and hazard changes, eliminations, and the odd body or health exception (hazard damage).
That is zstd-compressed, with a dictionary trained on the games once 2,000 are stored. Every
game is decoded and compared with the engine's frames before it is saved; one that doesn't
reproduce exactly is stored as raw JSON instead (format 0). Dropped: latency, shouts and
timing; kept: bodies, health, eliminations (cause, turn, by whom), food, hazards, and each
snake's name, author, colour, head and tail.

## Using it

```sh
sudo bsscrape stats --db /var/lib/bsscrape/games.sqlite            # counts and sizes
sudo bsscrape export --db /var/lib/bsscrape/games.sqlite <game id>  # one game as JSON
sudo bsscrape dump --db /var/lib/bsscrape/games.sqlite > games.jsonl # all, for training
```

`game_snakes` (with `strings`) says who played what, e.g. every game by an author:

```sql
SELECT lower(hex(g.game)) FROM game_snakes g JOIN strings s ON s.id = g.author WHERE s.s = 'brensch';
```

## Deploying

```sh
deploy/create-vm.sh   # once: the VM, IAP-only SSH
deploy/deploy.sh      # build and install; re-run to update (the database is kept)
gcloud compute ssh bsscrape --project snake2-338906 --zone us-central1-a --tunnel-through-iap
sudo journalctl -u bsscrape -f
```
