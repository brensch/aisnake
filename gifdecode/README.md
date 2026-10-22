# gifdecode

Turns the game replay GIFs that `bucket.go` saves to `gs://gregorywebp` (Battlesnake's
exporter render: 224x224, 11x11 board) back into game states.

```sh
cargo build --release
./target/release/snakedecode gs://gregorywebp games.jsonl.zst summary.tsv   # on a GCE VM: streams from the bucket
./target/release/snakedecode ./some-gifs   games.jsonl.zst summary.tsv      # or a local folder of .gif
```

Output is one JSON object per game, zstd-compressed:

```json
{"id": "...", "w": 11, "h": 11, "colors": ["#00ff00", ...], "errors": 0,
 "turns": [{"food": [[x, y], ...], "snakes": [[[x, y], ...] | null, ...]}, ...]}
```

Each snake's body is head first, with stacked segments repeated (as in the Battlesnake API);
`null` once it is eliminated. Coordinates are Battlesnake's: x right, y up from the bottom row.
`summary.tsv` has one line per game with its turn count, snake count and error count.

## How it decodes

The GIF has one frame per turn. Each cell is sampled (6x6 pixels) and classified as snake
paint, food, a dead snake's grey body, or board (ignoring the faint logo watermark). Rather
than trusting custom head and tail art, it simulates the game:

- turn 0: every painted cell is a snake of length 3, stacked;
- each turn, every live snake moves into a neighbouring cell: the free cell that is painted,
  in its own colour, and changed most since last turn; ties are settled best-first across all
  snakes; a snake grows when it moved onto food;
- a snake is eliminated when its body turns grey (the exporter greys dead snakes).

## How it checks itself

Every turn the decoded state is compared with the picture. A cell is an error if it is
clearly painted but no snake covers it, if a snake covers bare board, or if a snake's middle
segment is painted in another snake's colour (nearest colour wins, so similar colours stay
apart). Dead snakes' grey bodies and the coloured head/tail art left on them are expected.
If a turn fails, every combination of candidate moves for that turn is tried and the one
that matches the picture is kept. `errors` is the number of mismatched cells over the game.

`tools/roundtrip.py` renders the original GIF next to a redraw from the decoded state, to
check by eye. `DEBUG=1` prints the first mismatched turn of each game and every elimination.
