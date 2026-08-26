# Relay — Phase 1 Spec: Tic-Tac-Toe (Hotseat, Server-Owned State)

## Context

Relay is a networked multiplayer game platform (tic-tac-toe and Four in a Row to start), built to
learn job-queue design, real-time networking, and load testing. The long-term architecture
includes WebSockets, matchmaking (direct invite + open queue), and a separate load-testing
service. **This spec covers Phase 1 only**: a single-process, single-game, no-networking
version that proves the game logic and UI work before multiplayer is added.

Do not build anything beyond what's described below. No WebSockets, no matchmaking, no auth,
no persistence, no framework dependencies.

## Goal

Two people can sit at one keyboard, share one browser tab, and play a full game of
tic-tac-toe — clicking alternating turns as X and O — with correct win/draw detection.
"Multiplayer" in this phase means hotseat, not networked.

## Architecture

- **Language:** Go, standard library only (`net/http`). No web framework.
- **Pattern:** the server owns all game state and logic. The client is a dumb renderer — it
  never computes whose turn it is, whether a move is legal, or who won. It only displays
  whatever JSON the server returns. This mirrors the state-push pattern Phase 2 will use over
  WebSockets, so the rendering logic won't need to be rewritten later — only the transport
  (fetch → socket) changes.
- **State storage:** a single in-memory board, held in a package-level (or `main`-scoped)
  variable. One game exists at a time. Guard access with a `sync.Mutex` even though hotseat
  mode is unlikely to race — this is cheap insurance and matches the concurrency-safety habit
  the project will need later.

## File structure (deliberately flat — do not create more structure than this)

```
relay/
├── go.mod
├── main.go        — server setup, HTTP handlers, game state, game logic all together
└── index.html      — board UI (inline CSS/JS is fine, no build step)
```

Do not create `internal/`, `cmd/`, or separate packages yet. There's only one caller of the
game logic right now (the HTTP handlers in `main.go`); splitting it into a package is a Phase 2
concern, once a WebSocket handler becomes a second caller.

## Game logic requirements

- **Board:** 9 cells, indexed 0–8, left-to-right/top-to-bottom. Each cell is empty, X, or O.
- **Turn tracking:** X always moves first. Track whose turn it is.
- **Move validation**, in order — reject with a clear error if any fail:
  1. Position is in range 0–8
  2. Cell at that position is empty
  3. Game is still in progress (not already won or drawn)
- **Win detection:** check all 8 winning lines (3 rows, 3 columns, 2 diagonals) after every
  move. A line of all-X or all-O wins for that player.
- **Draw detection:** if all 9 cells are filled and no winner, the game is a draw.
- **Game result states:** in progress, X wins, O wins, draw.

## HTTP API

Both endpoints return the same JSON shape, so the client has a single render function.

### `GET /state`

Returns the current game state. No body required.

**Response 200:**
```json
{
  "board": ["", "X", "", "", "O", "", "", "", ""],
  "turn": "X",
  "result": "in_progress"
}
```
- `board`: array of 9 strings, each `""`, `"X"`, or `"O"`
- `turn`: `"X"` or `"O"` (whose turn is next; irrelevant once result != in_progress)
- `result`: one of `"in_progress"`, `"x_wins"`, `"o_wins"`, `"draw"`

### `POST /move`

**Request body:**
```json
{ "position": 4 }
```

**Response 200** (move accepted) — same shape as `GET /state`, reflecting the new state.

**Response 400** (move rejected — out of range, occupied cell, or game already over):
```json
{ "error": "cell already occupied" }
```
Use a distinct, human-readable message per rejection reason (out of range / occupied / game
over) so the frontend can surface something meaningful, but the client should not need to
parse the message to decide what to render — it just needs to know the move failed and
re-fetch or keep current state.

### `GET /` (or static file serving)

Serves `index.html`. A basic `http.ServeFile` or `http.FileServer` is sufficient — no routing
library needed.

### (Optional, nice-to-have) `POST /reset`

Resets the board to a fresh game. Not required for Phase 1 to be "done," but trivial to add
and useful for testing without restarting the server.

## Frontend (`index.html`)

- A 3×3 grid of clickable cells.
- A status line showing either whose turn it is, or the game result (e.g. "X's turn",
  "O wins!", "Draw").
- On page load: `fetch('/state')`, render the response.
- On cell click: `fetch('/move', { method: 'POST', body: JSON.stringify({ position: n }) })`,
  then re-render from the response (whether success or error). If the response is an error,
  show it somewhere visible (e.g. briefly in the status line) rather than failing silently.
- No frameworks, no build step. Plain HTML/CSS/JS is sufficient and expected.
- Clicking an already-filled cell, or clicking after the game is over, should be handled
  gracefully — either disable those cells client-side as a nicety, or rely on the server's
  400 response and just not update the board. Either is acceptable; server-side validation is
  the source of truth either way.

## Definition of done

- `go run main.go` starts a server that serves the board at `/`.
- Two people can click alternating cells in one browser tab and complete a full game.
- Wins are correctly detected on all 8 lines; draws are correctly detected.
- Invalid moves (occupied cell, out-of-range, move after game over) are rejected with a 400
  and do not corrupt state.
- No WebSockets, no second game, no auth, no persistence — none of that is in scope here.

## Explicitly out of scope for this phase

- WebSockets / real networking between two separate clients
- Matchmaking (direct invite or queue)
- Auth (Clerk or otherwise)
- Four in a Row
- Any package structure beyond `main.go` + `index.html`
- Any JS framework or CSS framework
- Persistence beyond the single in-memory game
