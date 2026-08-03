# Relay

A networked multiplayer game platform, starting with tic-tac-toe, built to learn job-queue
design, real-time networking, and load testing. The long-term architecture includes
WebSockets, matchmaking (direct invite + open queue), and a separate load-testing service.
This repo currently implements Phase 1 — see [Roadmap](#roadmap) for what that means and
what's not built yet.

## Running it

```sh
go run main.go
```

Then open `http://localhost:8080`.

## API

| Endpoint      | Method | Description                                              |
|---------------|--------|-----------------------------------------------------------|
| `/`           | GET    | Serves the board UI (`index.html`)                        |
| `/state`      | GET    | Returns the current game state                            |
| `/move`       | POST   | Applies a move: `{"position": 0-8}`                       |
| `/reset`      | POST   | Resets the board to a fresh game                          |

`GET /state` and `POST /move` return the same JSON shape:

```json
{
  "board": ["", "X", "", "", "O", "", "", "", ""],
  "turn": "X",
  "result": "in_progress"
}
```

`result` is one of `in_progress`, `x_wins`, `o_wins`, `draw`. Rejected moves (out of range,
occupied cell, or game already over) return `400` with `{"error": "<reason>"}`.

## Architecture

- Go standard library only (`net/http`) — no web framework.
- Game state (`board`, `turn`) and a `sync.Mutex` live together on a single `GameState`
  struct, bundled so the mutex's purpose is self-documenting rather than inferred.
- Win/draw detection (`computeResult`) is a pure read over the board — it never locks, so it's
  safe to call from methods that already hold the lock (like `MakeMove`).
- Handlers are methods on `*GameState` rather than closures over a package-level global, which
  keeps each HTTP test able to construct its own isolated game instance.

## Testing

```sh
go test ./...
go test -race ./...
```

Covers win/draw detection across all 8 lines, move validation and turn tracking, HTTP handler
behavior via `httptest`, and reset.

## Roadmap

This repo currently implements **Phase 1**: a single-process, single-game, no-networking
version that proves the game logic and UI work before multiplayer is added. Two people sit at
one keyboard, share one browser tab, and play a full game of tic-tac-toe — clicking alternating
turns as X and O.

The server owns all game state and logic; the client only renders whatever JSON the server
returns. This mirrors the state-push pattern later phases will use over WebSockets, so the
rendering logic won't need to be rewritten — only the transport (fetch → socket) changes.

Not yet built: WebSockets, matchmaking, auth, additional games beyond tic-tac-toe, and
persistence beyond the single in-memory game. See `relayspec.md` for the full Phase 1 spec.
