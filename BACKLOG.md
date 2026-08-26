# Relay — Backlog

Open decisions and follow-ups noted during active work, not yet scoped to a phase. Distinct
from `specs/` (each phase's committed spec) — this is where undecided things go until they're
either actioned or folded into a spec.

## CI: local formatting enforcement — pre-commit hook vs. CI-only gate

**Context:** CI now fails the build on `gofmt` violations (`chore/add-gofmt-ci`), but that's a
backstop — it only catches unformatted code after it's pushed, not before it's committed.

Options:
- **Pre-commit hook**: block (or auto-fix) locally before a commit is even made, so CI rarely
  fails on formatting.
- **Status quo**: rely on the CI gate; run `gofmt -w .` manually when it fails.

**Undecided.** Revisit if CI formatting failures become frequent enough to be annoying.

## ~~Disconnect handling: forfeit countdown~~ — promoted into Phase 2

**Resolved into scope.** Originally deferred here on the assumption that reconnection required
user accounts. That turned out to be wrong: reconnection only needs an ephemeral, in-memory
session token scoped to the current game's lifetime — not persistent auth. Given that, Phase 2
now includes: disconnect starts a grace-period timer (not an immediate game-over), the
remaining player can manually reset early or wait for the timer, and the disconnected player
can reconnect within the grace period using their token to resume their role.

## Reconnect token delivery: query param vs. cookie (post-Phase 2 hardening)

**Context:** Phase 2 sends the reconnect token as a query param on the WebSocket upgrade
request (`GET /ws?token=...`), checked before the handshake completes — kept consistent with
how first/second-connect role assignment is already decided at upgrade time. Known trade-off:
query params can end up in access logs and browser history, unlike a cookie. Accepted for now
because the token is ephemeral (scoped to one in-memory game) and a real deployment would run
over `wss://`, so the risk is low at this project's current scale (no reverse proxy/CDN in
front of it yet).

**Real fix:** deliver the token via a cookie instead — cookies attach automatically to a
same-origin WS upgrade without appearing in the URL. Not done now because the browser
`WebSocket` API can't set custom headers on the handshake (URL and subprotocol list are the
only client-controllable pieces), so adopting a cookie means also adopting `Set-Cookie`
semantics and flags (`HttpOnly`, `Secure`, `SameSite`) — more machinery than this phase needs.

**Deferred.** Revisit if this moves behind a proxy/CDN or logging becomes a real concern.

## Hub broadcast writes bypass the write-pump pattern

**Context:** `broadcastCount` (`hub.go:62`) writes directly to each connection via `conn.Write()`.
Once per-connection write pumps (`outbox`/`done`, see `specs/phase-2.md`'s write pump shutdown
design) exist, nothing should write to a connection except that connection's own write pump —
direct writes from `Hub` bypass the backpressure/decoupling the pump architecture exists for.

**Deferred.** Revisit when wiring `writePump` in — `broadcastCount` (or whatever replaces it)
should push onto each `Client`'s `outbox` instead of writing directly.
