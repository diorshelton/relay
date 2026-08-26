# Relay — Roadmap

This project is being built in phases, sequenced so each one both ships something playable
and deliberately teaches a specific Go/systems-design concept, rather than piling on
everything at once. See [README.md](README.md) for what's runnable right now.

### Phase 1 — Hotseat, single process (done)

Single-process, single-game, no-networking. Two people share one browser tab and click
alternating turns as X and O. The server owns all game state and logic; the client only
renders whatever JSON it's given — a state-push pattern that carries forward unchanged into
later phases. See `specs/phase-1.md`.

### Phase 2 — Networked multiplayer over WebSockets (in progress)

Two players in separate browser tabs play over a live WebSocket connection instead of sharing
a tab. Introduces the read-pump/write-pump-per-connection pattern, latest-value-wins
backpressure on state pushes, and an ephemeral reconnect-token scheme with a 20-second grace
period on disconnect. See `specs/phase-2.md`.

### Phase 3 — Multi-game support, matchmaking, Grid Drop

- Generalizes the single global `GameState` into a registry (`map[gameID]*GameState`), with
  its own mutex guarding only the map — not the games inside it, to avoid serializing every
  game behind one lock.
- A matchmaking queue keyed by game type from the start (even with only two types registered),
  so a Grid Drop player can never be paired with a tic-tac-toe player, and adding a third game
  type later doesn't force a redesign.
- Grid Drop as a second turn-based game, implemented against a generalized `Game` interface.
  Chosen specifically to validate that abstraction (registry + matcher + interface) using a
  game shaped like the one that already works, before Phase 5 also has to solve real-time
  state ownership at the same time.
- Protocol versioning (`v` field on the WebSocket message envelope) and lightweight structured
  logging (`slog`, with connection/game IDs) — both pulled forward into this phase rather than
  Phase 4, since they don't depend on concurrency pressure to be worth writing and only get
  more annoying to retrofit as more message types and entities accumulate.

### Phase 4 — Monitoring & resilience

- Metrics (active games, matchmaking queue depth per type, time-to-match) — deferred until
  after Phase 3 on purpose, since a metrics taxonomy only makes sense once there's an actual
  registry and matcher producing variance under concurrency to observe.
- `context.Context` propagation and a real cancellation tree per game/connection — also
  deferred, since there's nothing meaningful to cancel independently until Phase 3's
  per-game goroutines exist.
- Graceful shutdown on `SIGTERM`, draining connections instead of dying mid-game.
- Fault-injection tests: killed connections mid-write, malformed input, slow readers, verified
  leak-free under `-race` with `goleak`. Placed after instrumentation is in place, so failures
  can be observed rather than guessed at.

### Phase 5 — Real-time game loop (Pong or Snake)

- The first genuinely continuous game: state changes on a fixed tick, not just on player
  input, requiring an actual game loop (`time.Ticker` + `select` multiplexing tick, input, and
  shutdown channels).
- State ownership shifts from the mutex pattern used everywhere else to an actor-style
  design — a single tick-loop goroutine is the sole owner of simulation state, with all
  reads/writes arriving over channels. Deliberately built the other way from Phase 1–3's
  shared-memory-plus-mutex approach, to compare both patterns directly.
- Ships with Phase 4's metrics/logging already in place, so tick-duration histograms and
  dropped-frame counts (from the existing backpressure slot) exist from the first commit.
- Tick-loop logic tested deterministically via `testing/synctest` (or an injected clock),
  rather than relying on real `time.Sleep`-based tests.

### Phase 6 — Load-testing service

- A separate program that actually backs up the project's stated goal: a worker pool of
  simulated WebSocket clients, ramped up via `errgroup` and a rate limiter, capturing p50/p99
  latency into a histogram.
- Results (concurrent-game ceiling, latency under load, where the tick loop starts to degrade)
  published in this README rather than left as an unverified design claim.
- Doubles as fault injection at scale against everything built in Phases 3–5.

### Later, bolted on — user accounts

Persistent user accounts via a third-party auth provider. Deliberately not sequenced into the
phases above — nothing in Phases 3–6 depends on identity existing, so this can be added
whenever it's wanted without forcing a redesign.

### Explicitly out of scope until their phase arrives

WebSockets and matchmaking are covered above once their phase lands; until then, nothing
beyond the current phase's scope is built — see the relevant `specs/phase-N.md` for each
phase's explicit "out of scope" list.
