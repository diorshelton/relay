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

## Next feature: Player identity & reconnection — scope and open alignment issues

**Context:** Scoped as the next feature branch after the Phase 2 scaffolding PR (#2). Implements
the token-based reconnection design already specified in `specs/phase-2.md`'s "Session & role
assignment" and "Disconnect & reconnection" sections. Recorded here rather than folded straight
into a branch because scoping it surfaced open questions that need resolving before, or while,
building it.

**Scope, as given:**

1. Token generation & role assignment
   - [ ] On successful role assignment for a fresh connection, generate an opaque random session
     token (UUID)
   - [ ] Store it in an in-memory map: token → role, scoped to the current game's lifetime
   - [ ] Send the token to the client as the first message on the open socket:
     `{type: "token", token: "..."}`
2. Client-side token storage
   - [ ] On receiving the `{type: "token"}` message, write the token to storage
   - [ ] On page load, before opening the socket, check storage for an existing token
   - [ ] If present, append it as a query param on the WS upgrade request: `GET /ws?token=...`
   - [ ] If absent, connect fresh with no token param
3. Reconnection validation (server)
   - [ ] On upgrade request, check for a token query param
   - [ ] Validate the token against the in-memory map — same decision point as fresh-connection
     role assignment, not a separate path
   - [ ] Valid token, role in disconnect grace period: resume that role, cancel the grace-period
     timer
   - [ ] Invalid, expired, or already-resumed token: reject at the HTTP layer (403) before
     completing the handshake
4. Disconnect handling
   - [ ] On socket close, start a grace-period timer for that role (rather than immediately
     freeing it)
   - [ ] Confirm/define grace-period duration
   - [ ] On grace-period expiry with no reconnect, free the role/token for reassignment
5. Cleanup
   - [ ] Decide and implement: clear token from client storage on game end (win/draw) — vs.
     leaving it stale
   - [ ] Decide and implement: clear/invalidate token from server's in-memory map at game end

**Open alignment issues surfaced during scoping (resolve before/while implementing):**

- **Prerequisite from `HANDOFF.md`:** three `Client`/`Join` design decisions were left open
  before item 1's role-assignment work can start — `role`'s type (bare `string` vs. a typed
  enum), how `Client` reaches `GameState` (stored field vs. a parameter to pump methods), and
  `Join`'s exact signature/behavior.
- **Item 2 storage mechanism:** the scope above says `localStorage`; `specs/phase-2.md:174`
  specified an in-memory JS variable, "for the tab's session," never persisted. `localStorage`
  is shared across every same-origin tab, so two players' tabs in the same browser would collide
  on the same key. Needs a decision: is surviving a page reload an intentional scope increase
  over the spec, and if so, is `sessionStorage` (per-tab) the right primitive instead of
  `localStorage`?
- **Item 4 grace-period duration:** `specs/phase-2.md:125` already fixed this at 20 seconds.
  Confirm this task reuses that value rather than reopening it.

**Undecided.** The three `Client`/`Join` items block starting this feature branch. The storage
and duration questions are smaller but shouldn't be silently defaulted during implementation —
resolve explicitly, even if quickly.
