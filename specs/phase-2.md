# Relay — Phase 2 Spec: Networked Multiplayer (WebSockets, Single Game)

## Context

Phase 1 proved out the game logic and UI in a single process, no networking, hotseat mode.
Phase 2 introduces real networking: two people on two separate clients (different browser tabs,
different machines) play one game of tic-tac-toe together over a live WebSocket connection,
instead of sharing one browser tab. This is the phase where the project's actual learning goals
— full-duplex networking, concurrency, backpressure — start to apply.

Discovery (how two people find each other) is explicitly not solved here — connecting to the
server is joining the one game that exists. Matchmaking, multiple concurrent games, and auth are
later concerns; see "Explicitly out of scope" below.

## Goal

Two players, each in their own browser, play a full game of tic-tac-toe together in real time —
the server pushes state changes as they happen instead of the client polling for them. If a
player's connection drops mid-game, the game doesn't end immediately: a grace period gives them
a chance to reconnect and resume their role before the game resets.

## Explicitly out of scope for this phase

- Matchmaking (direct invite or open queue) — no way to discover an opponent beyond both people
  connecting to the same server
- Auth / user accounts — reconnection uses an ephemeral, in-memory session token scoped to the
  current game only, not a persistent identity
- Multiple concurrent games (`map[gameID]*GameState`) — still one global game at a time, same as
  Phase 1
- Spectators — a third connection attempt is rejected outright
- Four in a Row
- Persistence beyond the current game's in-memory lifetime
- Any JS framework or CSS framework (carried over from Phase 1)

## Architecture

- **Real WebSockets via `coder/websocket`** (`github.com/coder/websocket`), not Server-Sent
  Events, not a hand-rolled implementation. Chosen over `gorilla/websocket` for: continuous,
  focused maintenance (versus gorilla's archived-then-revived history), an idiomatic
  `context.Context`-native `Read`/`Write` API, and because it lets this project deliberately
  build a read-pump/write-pump architecture for backpressure, rather than have that
  architecture forced on it by a library correctness constraint (gorilla connections only
  support one concurrent writer; coder's `Write` is safe for concurrent use).
- **This is the first deliberate deviation from Phase 1's "standard library only" rule.** Go's
  standard library has no WebSocket implementation, so an external dependency is unavoidable for
  genuine full-duplex networking. Called out explicitly here rather than left as a silent
  `go.mod` change.
- **State model:** still a single, global, in-memory `GameState`, guarded by its existing
  `sync.Mutex` — no multi-game support yet. The seam for `map[gameID]*GameState` remains
  available for a future phase but isn't built now.
- **Read pump:** one goroutine per connection, blocked in a loop on `conn.Read(ctx)`, parsing
  and dispatching incoming messages (moves, resets, reconnect attempts).
- **Write pump:** one goroutine per connection, chosen deliberately even though `Write` is
  already safe for concurrent use — its purpose here is backpressure/decoupling, not
  correctness: the code that produces a new game state should never block on a slow client's
  socket write.
- **Backpressure policy:** latest-value-wins, not a FIFO-buffered channel. State pushes are full
  snapshots (`{board, turn, result}`), so a value queued behind a slow write is worthless the
  moment a newer one exists — a size-1 "latest state" slot (drop-and-replace) fits better than
  an unbounded or fixed-size buffer.
- **Encapsulation:** the read pump, write pump, and slot/channel machinery live inside a small
  per-connection type (e.g. `Client`), exposing a minimal public surface (e.g. `Send(state)`).
  Game logic never touches goroutines or channels directly — mirrors Phase 1's existing pattern
  of keeping transport concerns behind methods on `*GameState`.
- **Package structure:** left as a judgment call at implementation time, not decided here. Phase
  1 named "a WebSocket handler becoming a second caller of the game logic" as the trigger for
  splitting into packages; since `/move` and `/state` are being removed rather than kept
  alongside the socket, that specific trigger isn't strictly met — but the codebase is growing
  enough (connection handling, pumps, token management) that flat `main.go` may or may not still
  be the right call.

## File structure

```
relay/
├── go.mod
├── go.sum         — new: dependency lockfile, once coder/websocket is added
├── main.go        — server setup, HTTP + WS handlers, game state, game logic, connection/pump handling
└── index.html      — board UI, updated to speak the WS message protocol instead of fetch
```

## Session & role assignment

- The server accepts WebSocket connections at `GET /ws`.
- **Fresh connection** (no `token` query param, or one that doesn't match a live session):
  assigned a role by connection order — first connection to claim a role becomes X, second
  becomes O. A third connection attempt with no valid reconnect token is rejected outright at
  the HTTP layer, before the handshake completes (no spectators in this phase).
- On successful role assignment for a fresh connection, the server generates an opaque random
  session token (e.g. a UUID), stores it in an in-memory map (`token → role`) scoped to the
  current game's lifetime, and sends it to the client as the first message on the open socket
  (`type: "token"`). It is never persisted beyond the current game and is unrelated to any
  account system.
- **Reconnection attempt:** the client presents its token as a query parameter on the upgrade
  request (`GET /ws?token=...`). The server validates it before completing the handshake — the
  same decision point as fresh-connection role assignment, kept consistent rather than split
  across two different moments in the connection lifecycle.
  - Valid token matching a role currently in its disconnect grace period: the connection resumes
    that role, and the grace-period timer is cancelled.
  - Invalid, expired, or already-resumed token: rejected at the HTTP layer (e.g. `403`) before
    the handshake completes.

## Disconnect & reconnection

- When a connected player's socket closes unexpectedly, the game does **not** end immediately.
  The server starts a 20-second grace-period timer.
- While the timer is running:
  - The remaining connected player can send a `reset` message at any time to skip the countdown
    and start a fresh game immediately.
  - If the disconnected player reconnects with a valid token before the timer elapses, they
    resume their role and play continues from the current state — the timer is cancelled.
- If the timer elapses with no reconnection and no manual reset, the server resets the game
  automatically: the board clears, both tokens are invalidated, and both connection slots free
  up for new fresh connections.
- A general `reset` message is available at any time during a game, not only during a grace
  period — carried over from Phase 1's optional `POST /reset`, useful for testing without
  restarting the server.

## Network API

`GET /` still serves `index.html`, unchanged from Phase 1. `GET /state`, `POST /move`, and
`POST /reset` from Phase 1 are removed — all gameplay now happens over the socket at `GET /ws`.

All messages on the open socket share a typed envelope:

```json
{ "type": "...", "payload": { ... } }
```

**Server → client:**

- `state` — full game state, same shape as Phase 1's `GET /state` response:
  `{"board": ["", "X", ...], "turn": "X", "result": "in_progress"}`. Pushed immediately on a
  successful connect/reconnect, and after every accepted move or reset.
- `token` — `{"token": "..."}`. Sent once, immediately after a fresh connection is assigned a
  role. Not resent on reconnect (the client already holds it).
- `error` — `{"message": "..."}`. Sent when a move or action is rejected (out of range, occupied
  cell, game already over, wrong turn for this connection's role).
- `opponent_status` — `{"connected": false, "grace_period_seconds": 20}` sent the moment the
  opponent's socket drops, so the remaining client's UI can show a "opponent disconnected,
  resetting in 20s" countdown. `{"connected": true}` sent if they reconnect within the window
  (`grace_period_seconds` omitted — the countdown is simply cancelled client-side).

**Client → server:**

- `move` — `{"position": n}`.
- `reset` — no payload. Skips a running grace-period countdown, or resets an in-progress or
  finished game at any other time.

## Frontend

- Replace `fetch('/state')` / `fetch('/move', ...)` with a single `WebSocket` connection to
  `/ws`.
- On connect: render whatever `state` message arrives; store the `token` message's value in
  memory (e.g. a JS variable) for the tab's session, so a dropped connection can be retried with
  the same token.
- On `error`: surface it the same way Phase 1 did — briefly in the status line, don't fail
  silently.
- On `opponent_status` (disconnected): show a countdown / "waiting for opponent to reconnect"
  message.
- No frameworks, no build step — same constraint as Phase 1.

## Definition of done

- Two separate browser clients (different tabs or machines) connect to the server and play a
  full game of tic-tac-toe together in real time — moves and results appear on both sides
  without polling or a manual refresh.
- First connection becomes X, second becomes O; a third connection is rejected.
- Killing one client's connection mid-game does not immediately end the game; the other client
  sees a grace-period countdown.
- Reconnecting with the same token within the grace period resumes the same role and the game
  continues from where it left off.
- Letting the grace period lapse (or the remaining player manually resetting) clears the board
  and frees both slots for a new game.
- `go vet`, `gofmt`, and `go test -race` all stay clean with the new WebSocket code.
