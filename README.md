# Relay

[![CI](https://github.com/diorshelton/relay/actions/workflows/ci.yml/badge.svg)](https://github.com/diorshelton/relay/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/diorshelton/relay)](go.mod)
[![License: MIT](https://img.shields.io/github/license/diorshelton/relay)](LICENSE)

A networked multiplayer game platform, starting with tic-tac-toe, built to learn job-queue
design, real-time networking, and load testing. The long-term architecture includes
WebSockets, matchmaking (direct invite + open queue), multiple games, and a separate
load-testing service.

**Status:** Phase 1 (hotseat, single process) is complete. Phase 2 (networked multiplayer over
WebSockets) is in progress — `/ws` currently only tracks and broadcasts a connection count;
real gameplay still runs over the Phase 1 REST endpoints below. See [ROADMAP.md](ROADMAP.md)
for the full phase plan and what's not built yet.

## Running it

```sh
go run main.go
```

Then open `http://localhost:8080`.

## API

| Endpoint | Method | Description                                                                                            |
|----------|--------|----------------------------------------------------------------------------------------------------------|
| `/`      | GET    | Serves the board UI (`index.html`)                                                                       |
| `/state` | GET    | Returns the current game state                                                                           |
| `/move`  | POST   | Applies a move: `{"position": 0-8}`                                                                      |
| `/reset` | POST   | Resets the board to a fresh game                                                                         |
| `/ws`    | GET    | WebSocket upgrade — scaffolding only for now (broadcasts a connection count), not yet wired to gameplay. See `specs/phase-2.md` for the target design. |

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

- Go standard library only (`net/http`) through Phase 1 — Phase 2 adds `coder/websocket` as
  the first external dependency. The standard library itself has no WebSocket support, and
  `golang.org/x/net/websocket` (the closest thing to one) is deprecated and unmaintained, so a
  maintained third-party package was the only real option.
- Game logic (`GameState`, move validation, win/draw detection) lives in `game.go`; connection
  tracking for the in-progress WebSocket work lives in `hub.go`; `main.go` wires both into HTTP
  handlers.
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

## Docs

- [ROADMAP.md](ROADMAP.md) — the phased build plan (1–6) and the reasoning behind the sequencing.
- `specs/` — the committed spec for each completed or in-progress phase.
- [Issues](https://github.com/diorshelton/relay/issues) — open decisions and deferred
  follow-ups, tracked with the `decision` and `tech-debt` labels.
