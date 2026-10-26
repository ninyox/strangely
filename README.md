# Strangerly

**One click away from someone new.**

A mobile-first random video chat: choose a display name for the visit, allow your
camera, press **Start matching**, and you are paired with one other waiting
visitor for a one-to-one WebRTC call. No account, no email, no password, no
recording. Skip is locked for the first five seconds — enforced by the server,
not by the button — and Stop, mute, camera-off, reporting, and closing the tab
are never blocked.

```text
strg/
  frontend/     React Router SPA (TypeScript, Tailwind v4) — copy of documents/subll
  backend/      Go service (Gin, gorilla/websocket) — copy of documents/subllgo
  scripts/      e2e-matchmaking.mjs — scripted two-visitor check
  prompt.md     the build specification this project implements
  README.md     this file
```

## 1. How it works

```text
   Browser A                                     Browser B
   ┌──────────────┐        WebRTC media          ┌──────────────┐
   │ camera/mic   │◄──────── (DTLS-SRTP) ───────►│ camera/mic   │
   └──────┬───────┘        peer to peer          └──────┬───────┘
          │                                            │
          │  HTTPS: session, queue, skip, stop, report │
          │  WSS:  offer / answer / ICE + state events  │
          └────────────────┬───────────────────────────┘
                           ▼
                  ┌──────────────────────┐
                  │  Go backend (:8080)  │
                  │  sessions · queue    │
                  │  matches · reports   │
                  │  in-memory only      │
                  └──────────────────────┘
```

The server never sees, relays, or records audio or video. It carries the few
small messages needed to set up a direct connection, and it owns every rule:
who is waiting, who is matched with whom, when the match started, and whether a
Skip is allowed yet.

### Session and identity

Opening the site calls `POST /api/session`, which sets one opaque, `HttpOnly`,
`SameSite=Lax` cookie (`Secure` in production). Only a SHA-256 hash of the token
is kept in memory; the raw value exists solely in the browser. The display name
you type is a label for the visit — it is never treated as identity, never has
to be unique, and is validated identically on both sides (3–20 characters,
letters/digits/spaces/`. _ -`, no control characters).

### Call states

`idle → queued → connecting → connected → ended`, as reported by
`GET /api/matchmaking/status`. A match is `connecting` until both sides report
their peer connection is up (`ready` over the signaling socket), then `matched`.

## 2. Design system

**Art direction: "broadcast paper".** Warm editorial print meets a live control
surface. Paper stock, a high-contrast serif and hairline rules say *considered
and human*; the dark stage, mono timecode and signal rings say *live, now, one
moment only*. Strangerly is a random one-to-one video call, so that tension is
the brand — a passing human connection, framed with care.

Every rule below follows from it, and the whole system lives in
`frontend/app/app.css` as Tailwind v4 `@theme` tokens.

### Typeface roles

| Role | Face | Used for |
| --- | --- | --- |
| Editorial voice | **Fraunces** (variable, optical sizing left to the browser) | Hero, section headings, the partner's name, large numerals |
| Interface voice | **Inter** (variable) | Body copy, labels, buttons, forms |
| Machine voice | system mono stack | Timecodes, state labels, kickers, the countdown |

All three are self-hosted and subset to latin + latin-ext: **208 KB of woff2**
replacing the 1.4 MB of static TTFs the project started with, with no
third-party request, so the Content Security Policy stays strict. The two latin
faces are preloaded.

### Colour, by role

| Token | Value | Meaning |
| --- | --- | --- |
| `paper` / `surface` / `clay` | `#fbfaf7` / `#fff` / `#f2ede6` | Page, raised, tinted band |
| `ink` / `ink-soft` / `ink-muted` | `#131211` / `#4a4744` / `#7a7570` | Text: 17.7:1, 9.4:1, 5.5:1 on paper |
| `hairline` / `hairline-strong` | `#e6e1da` / `#d5cfc6` | Structure, carried by 1px rules |
| `signal` / `signal-bright` | `#d92d20` / `#f04438` | Action and liveness |
| `stage` / `stage-raised` / `stage-line` | `#0e0d0c` / `#1c1a18` / `#302c28` | The live call surface |
| `success` / `warning` / `danger` | `#067647` / `#b54708` / `#b42318` | Status |

The action colour is `#d92d20` rather than `#f04438` because white text on it
measures **4.83:1** (WCAG AA for normal text); `#f04438` is kept for rings, dots
and focus, where the 3:1 non-text threshold applies. A script checked every
pairing rather than trusting the eye.

### Composition

Four movements on the landing page, and no two adjacent sections share a tone or
a layout: **paper hero → dark stage → clay safety → signal closing → paper
footer**. Structure comes from hairline rules, not from a card in every slot;
shadows appear only where something is genuinely lifted off the page. Radii are
role-based (4px chips, 8px controls, 14px surfaces, 22px stage), so nothing is
rounded by reflex. The product is its own imagery: the live camera monitor in
the hero and the call stage in the dark band are the real components in their
empty states, not mock-ups.

### Motion

One curve (`cubic-bezier(0.22, 1, 0.36, 1)`), four durations, and four named
animations: scroll reveal (14px rise, staggered, once), the searching rings, the
live dot, and the connecting sheen. `prefers-reduced-motion` switches all of it
off, and nothing animates on a property that triggers layout.

### Responsiveness

Fluid `clamp()` type means no breakpoint juggling; the layout grid moves from one
column to a deliberate asymmetry at `lg`. The call screen becomes a dark room
with a **sticky bottom control bar** on a phone — thumb-reachable, 44px targets,
`env(safe-area-inset-bottom)` padding — and dialogs become bottom sheets below
`sm`. Tap targets, not just text, scale. The call itself can be expanded to fill
the screen, by the control on the picture; the control bar stays on screen in
that mode, so Stop is never the thing that gets hidden.



| Tool | Version used | Notes |
| --- | --- | --- |
| Go | 1.26.5 | backend (`go.mod` requires 1.26.5) |
| bun | 1.3.x | frontend install, dev server, tests |
| Node | 24.x | production static server (`frontend/server.mjs`), e2e script |

No database, no Redis, no external service is required for the local
prototype.

## 4. Local development

Two terminals.

**Backend**

```bash
cd backend
cp .env.example .env          # optional: every value has a working default
go run .                      # http://localhost:8080
```

**Frontend**

```bash
cd frontend
bun install
cp .env.example .env.local    # optional: defaults to http://localhost:8080
bun run dev                   # http://localhost:5173
```

Open <http://localhost:5173>, type a name, allow the camera, press **Start
matching**. Open a second browser profile (or a private window) and do the same
to be matched with yourself.

The backend allows `http://localhost:5173` by default. If you change the
frontend port, set `ALLOWED_ORIGINS` to match — it is the allowlist for CORS
**and** for the signaling WebSocket `Origin` check.

## 5. API

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/session` | cookie (optional) | Create or resume the anonymous session; sets the cookie |
| `GET` | `/api/session` | session | Read the current session |
| `GET` | `/api/config` | public | ICE servers, Skip window, report reasons |
| `POST` | `/api/matchmaking/join` | session | Enqueue (or be matched immediately) |
| `POST` | `/api/matchmaking/cancel` | session | Leave the queue |
| `GET` | `/api/matchmaking/status` | session | Current state, match, and remaining Skip lock |
| `POST` | `/api/matches/{id}/skip` | session | End the call — `409 skip_locked` inside the window |
| `POST` | `/api/matches/{id}/stop` | session | End the call immediately, always allowed |
| `POST` | `/api/matches/{id}/report` | session | Report the call (never blocked by the Skip lock) |
| `GET` | `/api/signaling` | session + Origin | WebSocket for offer/answer/ICE and state events |
| `GET` | `/healthz` | public | Process is up |
| `GET` | `/readyz` | public | Ready for traffic (503 while draining) |

Errors use one envelope: `{ "success": false, "code": "skip_locked", "message": "…", "data": { … } }`.
The `code` is the stable contract the UI reacts to; the message is copy for
humans.

Signaling messages (both directions) are
`{ "type": "offer" | "answer" | "ice" | "ready" | "bye" | "ping", "matchId": "…", "payload": { … } }`,
and the server pushes `hello`, `matched`, `partner_ready`, `partner_left`,
`peer_unreachable`, `pong`, and `error`.

Three details of that contract matter in a real deployment:

* **A `bye` may carry `{ "reason": "disconnected" }`.** The server accepts that
  one reason (anything else is treated as `left`), so a browser that gives up on
  a peer connection that never came up tells the partner *why* instead of
  leaving it in a call that cannot work.
* **`error` with `code: "partner_offline"`** means a signaling message was not
  delivered because the partner has no live socket. The sending side is told
  immediately rather than waiting for its own ICE to time out.
* **The greeting is authoritative.** `hello.state` reports the session's real
  state, so a client that reconnects either resumes the call it is in, ends a
  call the server has already closed, or says goodbye to a call it never joined.
  Matchmaking also waits for the signaling socket to open before joining the
  queue: a session the server cannot reach must never be paired with somebody.

## 6. The five second Skip rule

`matchedAt` is recorded on the server when two sessions are paired. A Skip
request before `matchedAt + 5s` is refused with `409 skip_locked` and the
remaining milliseconds:

```json
{ "success": false, "code": "skip_locked", "message": "Skip is available in 4 seconds.",
  "data": { "matchId": "…", "remainingMs": 3874, "skipLockMs": 5000, "serverTime": "…" } }
```

The countdown in the UI is presentation only. It is recomputed from `matchedAt`
plus the offset between the server's `serverTime` and the browser's clock, and
any refusal from the server overrides the local estimate — so a wrong, skewed,
or deliberately moved browser clock changes nothing.

The rule applies to **Skip only**:

| Action | During the first 5 seconds |
| --- | --- |
| Skip | Refused by the server (`409 skip_locked`) |
| Stop | Allowed |
| Mute / camera off | Allowed |
| Report | Allowed |
| Closing the tab | Allowed (the partner sees "Your partner left") |

Once the window has passed, a Skip ends the call for both participants, notifies
the partner (`partner_left`, reason `skipped`), tears down the peer connection
and signaling state, and shows the other person a clean end state with **Find
someone new**.

## 7. Safety, privacy, and abuse handling

* **Community guidelines** are shown before starting and on `/guidelines`.
* **Report** is available for the entire call and for a moment after it ends. A
  report stores the reason, an optional note, the match, and the time — no
  video, no audio, no name, no IP address.
* **Stop** is never gated.
* **Rate limits** apply to session creation (per address), joining (per
  session), signaling messages (per session per second), and reporting (5 per
  10 minutes per session).
* **Validation**: usernames and report reasons are validated on both sides;
  request bodies are capped at 8 KiB; signaling messages at 64 KiB; malformed
  JSON is rejected; unknown signaling types are refused.
* **Cross-session signaling is impossible**: every relay is checked against the
  sender's own active match, and match ids are unguessable.
* **CORS** is restricted to the configured origin list with credentials enabled
  — never a wildcard. The WebSocket handshake validates `Origin` against the
  same list.
* **Security headers** are set on both the API and the static app, including a
  Content Security Policy. The app's CSP uses a per-response nonce for the SPA
  shell's inline bootstrap scripts, so `script-src` never falls back to
  `'unsafe-inline'`.
* **No IP addresses** are written into session records or logs. The address is
  used only as a rate-limit key, in memory. Behind a reverse proxy or a hosting
  provider, that layer may keep its own access logs — see `/privacy`.
* **No moderation in real time, no identity or age verification.** The UI says
  so, and does not claim otherwise.


## 8. Configuration

Backend (`backend/.env.example`, loaded from `.env` by `config.LoadEnv`; a
`.env.local` beside it is read first and wins, for a development machine only):

| Variable | Default | Purpose |
| --- | --- | --- |
| `ENVIRONMENT` | `development` | `production` switches to JSON logs |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | CORS + WebSocket `Origin` allowlist (comma separated) |
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `COOKIE_NAME` / `COOKIE_SECURE` / `COOKIE_DOMAIN` | `strg_session` / `false` / — | Session cookie |
| `SESSION_TTL` | `2h` | Idle session lifetime (slides on activity) |
| `QUEUE_TTL` | `2m` | How long a waiting entry survives |
| `MATCH_IDLE_TTL` | `20s` | Grace period before a silent peer's call is ended |
| `MAX_MATCH_DURATION` | `1h` | Hard cap on one call |
| `SKIP_LOCK_SECONDS` | `5` | The server-enforced Skip window |
| `STUN_URLS` | `stun:stun.l.google.com:19302` | ICE STUN servers (comma separated) |
| `TURN_URLS` | — | TURN URLs (comma separated; UDP, TCP and TLS entries share one credential). `TURN_URL` is still read as the single-URL form |
| `TURN_SHARED_SECRET`, `TURN_CREDENTIAL_TTL` | — / `1h` | TURN with short-lived credentials minted per request |
| `TURN_USERNAME`, `TURN_CREDENTIAL` | — | Static TURN credentials for a hosted relay (handed to every visitor) |
| `RATE_LIMIT_SESSION_PER_MINUTE` etc. | `20` / `60` / `60` / `5` | Session, join, signaling, report budgets |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `TRUSTED_PROXIES` | empty | Proxy addresses; empty means use the socket peer address |

Durations accept a bare number of seconds (`5`) or a Go duration (`2h`).
Nothing reads a secret at build time, and no real credential is committed.

Frontend (`frontend/.env.example`):

| Variable | Default | Purpose |
| --- | --- | --- |
| `VITE_API_BASE_URL` | `http://localhost:8080` (dev) | The single API base URL; the WebSocket URL and the CSP `connect-src` are derived from it. **Baked into the bundle at build time** — set it for the production build, or the build falls back to the origin it is served from |
| `PORT` | `3000` | Port for the production static server |

## 9. Tests

**Backend** (unit + HTTP + WebSocket integration, against the real router):

```bash
cd backend
go vet ./...
go test ./...            # add -race for the concurrency check
```

Covers: opaque hashed session tokens, TTL and sliding expiry, username
validation, two waiting sessions matched exactly once, no self-match, no
duplicate queue entry, no join while in a match, FIFO pairing, stale queue
cleanup, partner disconnect and abandoned-match cleanup, the Skip lock (refused
inside the window with the remaining time, allowed after, idempotent), Stop
inside the window, report scoping and rate limiting, cross-session signaling
refusal, `Origin` rejection on the socket, oversized and malformed messages,
malformed and oversized bodies, rate limits, `/healthz`, `/readyz`, and the
public config payload.

**Frontend** (pure logic, no DOM needed):

```bash
cd frontend
bun run typecheck
bun test
bun run build
```

Covers the username rules (rune-based length, control characters, non-Latin
scripts) and the Skip countdown maths (including a browser clock that is two
minutes wrong).

**End to end** — two simulated visitors, two cookie jars, two real signaling
sockets, a real SDP/ICE exchange, against a running backend:

```bash
cd backend && SKIP_LOCK_SECONDS=3 go run . &
cd .. && node scripts/e2e-matchmaking.mjs
```

It asserts 46 behaviours: session creation and resume, invalid-name rejection,
anonymous access refusal, queue → match → matched events on both sides,
initiator selection, offer/answer/ICE relay, readiness flipping the state to
`matched`, the Skip refusal with its remaining time, reporting inside the lock,
an outsider being unable to end or signal into a call, Stop inside the lock,
idempotency, a fresh match afterwards, and a Skip after the window ending the
call for both.


## 10. Deployment

**Build**

```bash
cd backend  && go build -o strgback .            # or: docker build -t strangerly-api .
cd frontend && bun install && bun run build      # or: docker build -t strangerly-web .
```

**Run**

```bash
cd backend  && ENVIRONMENT=production ALLOWED_ORIGINS=https://strangerly.example \
               COOKIE_SECURE=true LISTEN_ADDR=:8080 ./strgback
cd frontend && VITE_API_BASE_URL=https://api.strangerly.example PORT=3000 node server.mjs
```

Both projects ship a `Dockerfile`. The frontend image takes
`--build-arg VITE_API_BASE_URL=…` because that value is baked into the bundle;
`server.mjs` reads the same variable at runtime for its CSP `connect-src`.

**Reverse proxy.** Caddy is the simplest option: `reverse_proxy` passes
WebSocket upgrades through untouched, so `/api/signaling` needs no special
handling at all.

```caddyfile
strangerly.example {
	encode zstd gzip

	# The API, /api/signaling included.
	handle /api/* {
		reverse_proxy 127.0.0.1:8080 {
			# Overwrite instead of trusting what arrived: by default Caddy
			# passes an incoming X-Forwarded-For through, which would let a
			# client choose the address the rate limit is keyed on.
			header_up X-Forwarded-For {remote_host}
		}
	}

	# The SPA: deep links fall back to the shell, so a hard reload of /chat
	# works. This serves build/client from disk, which is what the rsync
	# deploy writes.
	handle {
		root * /var/www/strangerly
		try_files {path} /index.html
		file_server
	}
}
```

Serving the files from disk costs the Content-Security-Policy: the nonce-based
policy lives in `frontend/server.mjs`, and Caddy cannot apply it to files it
serves itself. To keep it, run the static server and point Caddy at that instead
— `VITE_API_BASE_URL=… PORT=3000 node server.mjs`, then
`reverse_proxy 127.0.0.1:3000` for the catch-all `handle`.

*nginx*, if you prefer it, needs the upgrade plumbing spelled out:

```nginx
server {
  listen 443 ssl;
  server_name strangerly.example;

  location / {
    proxy_pass http://127.0.0.1:3000;      # frontend/server.mjs
  }

  location /api/ {
    proxy_pass http://127.0.0.1:8080;      # Go backend
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

    # WebSocket upgrade for /api/signaling
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 3600s;
  }
}
```

**Client addresses.** With a proxy in front, set `TRUSTED_PROXIES` to the address
the proxy connects *from* — `127.0.0.1` when it proxies to a loopback port, a
Docker bridge address when it runs in a container. Get this wrong and every
visitor looks like the proxy, which turns the per-address session limit into one
bucket shared by the whole world. The backend's own access log tells you what it
resolved: `journalctl -u strgback | grep /api/session` must print the visitor's
address, not the proxy's. Because both Caddy and nginx pass an incoming
`X-Forwarded-For` through by default, overwrite it when the proxy is the edge
(`header_up X-Forwarded-For {remote_host}` in Caddy,
`proxy_set_header X-Forwarded-For $remote_addr` in nginx) — otherwise a client
can forge the address the rate limit is keyed on.

Also set `ALLOWED_ORIGINS` to the public origin and `COOKIE_SECURE=true`.

**When two phones never connect, check these in order.** They can look alike
from the outside — a call that never starts, or one side told the other
"disconnected" although it never connected — so it is worth knowing which link
is broken:

1. **The signaling socket.** The backend log says which of the three it is, in
   one line each: `signaling refused: origin not allowed` (the `Origin` the
   browser sent is not in `ALLOWED_ORIGINS`), `signaling refused: no session
   cookie` (the cookie was not sent — app and API are on different *sites*), or
   `signaling upgrade failed` (an nginx proxy that is not forwarding
   `Upgrade`/`Connection`; Caddy handles upgrades on its own). `signaling
   connected` means the socket is fine, and
   `signaling read ended` / `signaling disconnected …` says when and why it
   stopped. In the browser, the WebSocket must reach `101 Switching Protocols`;
   the same checks by hand:
   `curl -i -H 'Origin: https://your.origin' -H 'Connection: Upgrade' -H 'Upgrade: websocket' https://your.api/api/signaling`.
2. **The session cookie.** The handshake also needs `strg_session`. Cross-*site*
   deployments (app and API on different registrable domains) do not send a
   `SameSite=Lax` cookie, so keep both on the same site (subdomains are fine).
   `COOKIE_SECURE=true` needs HTTPS end to end; an empty `COOKIE_DOMAIN` is the
   right choice and makes the cookie host-only.
3. **The media path**, when both phones reach the call screen, show their own
   camera, and stay on "Connecting" with a blank partner tile. That is the peer
   connection, not signaling. A browser that gives up now reports what it saw,
   and the backend logs it:
   `browser gave up on its peer connection … ice=failed local_candidates=8
   remote_candidates=0 remote_description=true`. Read it as:
   `remote_description=false` → the offer or answer never crossed (a signaling
   problem after all); `remote_candidates=0` → the partner's candidates never
   crossed (relay or firewall); candidates on both sides with `ice=failed` → the
   candidates crossed and never paired, which is NAT. On Android,
   `chrome://webrtc-internals` shows the same thing live, including the
   candidate pairs. `LOG_LEVEL=debug` also traces every relayed offer, answer and
   candidate, so a whole call can be followed in the log.
4. **TURN**, for the NAT case in 3: two mobile networks usually cannot connect
   with STUN alone, while two phones on the same Wi-Fi normally can — so test
   both phones on one Wi-Fi first to tell the two apart. Set `TURN_URLS` and
   either a shared secret or a static credential pair, and every visitor gets
   the relay in `GET /api/config`.

**Adding TURN (coturn).** TURN is the only fix for the case in 3 where the
candidates crossed and never paired, and it is what makes mobile-to-mobile calls
work. The backend already mints time-limited credentials from a shared secret
(`handlers/config.go`), so coturn only needs the same secret:

```ini
# /etc/turnserver.conf
listening-port=3478
fingerprint
use-auth-secret
static-auth-secret=REPLACE_WITH_A_LONG_RANDOM_SECRET
realm=serve.subll.org
min-port=49160
max-port=49200
no-multicast-peers
no-cli
```

```bash
# backend/.env
TURN_URLS=turn:serve.subll.org:3478?transport=udp,turn:serve.subll.org:3478?transport=tcp
TURN_SHARED_SECRET=REPLACE_WITH_A_LONG_RANDOM_SECRET   # identical to coturn
TURN_CREDENTIAL_TTL=1h
```

Every URL in `TURN_URLS` is handed to the browser as one entry with the same
credential, which is what makes a UDP entry, a TCP entry and a TLS-on-443 entry
work as fallbacks for each other.

Allow **3478/udp**, **3478/tcp** and the relay range (**49160-49200/udp**) in the
host firewall *and* in the provider's firewall — a blocked relay range is the
usual reason "TURN is configured" and calls still fail. For networks that block
UDP entirely, add a TLS listener (`tls-listening-port=5349` plus `cert`/`pkey`)
and list that URL as well: `TURN_URLS` takes all of them at once.

Verify: the startup log says `turn_configured=true` (with `turn_urls=N`), and
`curl -s https://your.api/api/config` lists a `turn:` entry with a `username`
and a `credential` that changes on every request. Then a call whose log line
shows `ice=connected` (or simply a picture) is one that used the relay.

`Your partner left` ("They closed the page") only ever comes from an explicit
goodbye, so it always means the other page went away: the tab was closed or
hidden, or the page was reloaded. On a phone that is usually the screen locking,
switching apps, or the OS discarding a backgrounded tab — worth knowing before
reading it as a network fault.

The app itself is now defensive about all three: it opens the signaling socket
*before* joining the queue, tells the partner when it gives up on a peer
connection, and reports `partner_offline` instead of silently dropping a
signaling message that could not be delivered.

**Scaling path (not required for the prototype)**

* **TURN**: a STUN-only setup fails on symmetric NATs and some corporate
  firewalls. Add a TURN server and configure `TURN_URLS` + `TURN_SHARED_SECRET`
  (or `TURN_URLS` + `TURN_USERNAME`/`TURN_CREDENTIAL` for a hosted relay); the
  backend then hands every visitor the relay, minting short-lived credentials
  per request in the shared-secret form, so no static credential is ever shipped
  to a browser.
* **Multiple replicas**: matchmaking state is per process, so two replicas
  cannot see each other's queues. Moving the `store.Store` methods onto Redis
  (queue, session records, match records, and pub/sub for signaling) is the
  designed seam; nothing else in the service needs to change.
* **Durable reports**: reports are kept in memory. A moderation workflow needs a
  datastore, and that is where retention rules belong.

## 11. Known limitations (honest list)

* **No database.** Sessions, queue entries, matches, and reports live in the
  memory of one process. Restarting the backend clears everything, and the
  service cannot run as multiple replicas without shared state.
* **STUN only by default.** Calls between two restrictive NATs will fail to
  connect without TURN. The UI shows a failed connection state rather than a
  frozen video, but the call itself cannot be rescued.
* **Reloading during a call ends it.** A peer connection cannot be restored from
  a cold start, so a reload says goodbye and the partner sees "Your partner
  left" instead of hanging.
* **No cross-device block list.** Without accounts there is nothing to attach
  one to, so a blocked person can be matched again in a later visit. The app
  says so rather than implying otherwise.
* **No moderation in real time.** Reports are stored for a human to review;
  there is no automated detection and no immediate removal.
* **No identity, age, or content verification.** Anyone can claim any name.
* **Camera and microphone only.** No text chat, no file sharing, and no
  recording — by design.
* **A missing `Origin` header is accepted on the signaling socket.** Browsers
  always send one, and a hostile page sends its own (which the allowlist
  rejects); allowing a missing header keeps CLI and load-test clients usable. If
  you never need that, require the header in `Hub.ServeWS`.
* **The mobile picture-in-picture tile is fixed in a corner.** It is not
  draggable, because a half-working drag interaction is worse than none.


## 12. Verification status

What was actually run, and what could not be:

* ✅ `go build ./...`, `go vet ./...`, `gofmt` clean.
* ✅ `go test ./...` — every package passes, including the WebSocket and HTTP
  integration tests.
* ✅ `go test -race ./...` — passes with no data races.
* ✅ `bun run typecheck`, `bun test` (22 assertions), `bun run build`.
* ✅ `node scripts/e2e-matchmaking.mjs` against a live backend: **46/46 checks
  passed** (queue, match, SDP/ICE relay, Skip refusal with remaining time,
  reporting inside the lock, outsider refusals, Stop inside the lock,
  idempotency, rematch, Skip after the window).
* ✅ **Rendered pages inspected in a real browser** (headless Chrome with a
  synthetic camera device) at 1440×900, 834×1112 and 390×844: no horizontal
  overflow at any width, no console errors, both fonts loading, all utilities
  generated, every scroll-reveal firing, and the local camera preview rendering
  live video (1280×720, two tracks). That pass caught two real defects that no
  amount of type checking would have found — a palette declared outside
  `@theme` so no colour utility existed (invisible primary buttons and dark
  bands), and a signaling greeting that a freshly matched client misread as a
  page reload, which made it hang up on its own partner. Both are fixed.
* ✅ The production static server serves the SPA, applies the CSP nonce to all
  three inline shell scripts, and answers `200` for deep links such as `/chat`.
* ✅ **Full screen driven end to end** in headless Chrome over the DevTools
  protocol: the browser is put into a real call (a scripted partner joins the
  queue), the control on the picture is pressed, and 16 assertions check that the
  call fills the viewport, the picture switches from `object-cover` to
  `object-contain`, **Stop stays inside the viewport** at 1280×757 and at
  390×844, nothing overflows sideways on a phone, leaving full screen restores
  the layout and the page scroll, and a call that ends while expanded gives the
  screen back. Screenshots at both sizes were inspected by eye.
  ⚠️ Native full screen itself could not be exercised: headless Chrome has no
  window manager, so `requestFullscreen()` never produces a fullscreen element
  and the **filled-page fallback** is what those runs verified — which is also
  the path an iPhone takes, since Safari there refuses the API for anything that
  is not a video element. The native path is a five-line branch of the same
  toggle; check it once by hand on a desktop browser.
* ❌ **Live peer-to-peer video between two browsers was not verified.** This
  environment has no camera and no display: the browser checks above used a
  synthetic device and confirmed that `getUserMedia` runs and that a video
  element renders real pixels locally, but the two-visitor call could not be
  driven to a connected peer connection here (the sandbox does not complete ICE
  gathering between headless contexts). The signaling protocol, ICE
  configuration path, and every state transition around them *are* verified by
  the scripted two-client check; the `RTCPeerConnection` wiring in
  `lib/webrtc.ts` is exercised only by type checking and by the browser
  handshake up to offer/answer relay. Treat the manual test below as the final
  acceptance step.

### Manual two-browser test

1. Open Strangerly in two separate browser profiles (or two devices).
2. Set different names and allow camera and microphone in both.
3. Press **Start matching** in both; they should match each other.
4. Confirm you can see and hear each other.
5. Try **Skip** immediately: it should refuse, showing `Skip in 5…4…3…2…1`.
6. Confirm **Stop**, **Report**, mute, and camera-off all work during those five
   seconds.
7. Press the full screen control on the picture: the call should fill the
   screen with the control bar still there, the picture letterboxed rather than
   cropped, and the control itself should turn into "leave full screen".
   Press it again (or `Esc`) and the page should come back exactly as it was.
8. After five seconds, Skip from one side and confirm the other side shows
   "Your partner left" with **Find someone new**.
9. Press **Find someone new** in both and confirm a fresh match.
10. Repeat with one tab closed mid-call, and with one side denying camera access
    (expect a clear permission message and no queue entry).

## 13. Changed files and run commands

**Run the whole thing**

```bash
# terminal 1
cd backend && go run .

# terminal 2
cd frontend && bun install && bun run dev     # http://localhost:5173
```

**Everything created or replaced** — only inside `frontend/` and `backend/`:

Backend, new: `main.go`; `config/env.go`;
`middleware/{session,security,ratelimit}.go`;
`models/models.go`;
`store/{store,sessions,join,queue,status,matches,reports,cleanup}.go`;
`handlers/{api,session,config,matchmaking,match,report,health,signaling}.go`;
`routes/routes.go`; `utils/{ids,username,response}.go`; `Dockerfile`;
`.env.example`; `.gitignore`; and tests in `utils/`, `store/`, `middleware/`,
`routes/`.
Backend, removed from the copy: `controller/` (Telegram bot and payments),
`email/`, `repositories/`, the Subll `models/` and `handlers/`,
`config/{bachs,bot,banks,database}.go`, `utils/money.go`,
`middleware/auth.go`, and the Subll notes.

Frontend, new: `app/app.css` (the whole design system: tokens, utilities,
motion), `app/root.tsx`, `app/routes.ts`,
`app/routes/{home,chat,privacy,guidelines,about,not-found}.tsx`,
`app/components/` (`wordmark` — the two-ring mark, `site-header`, `site-footer`,
`hero`, `live-panel`, `camera-preview`, `how-it-works`, `stage-preview`,
`safety-section`, `closing-cta`, `call-stage`, `call-controls`, `countdown-ring`,
`state-panels`, `report-dialog`, `safety-dialog`, `page-shell`, `reveal`,
`video-tile`, `notice`, `spinner` — spinner, signal rings and the live dot, and
`ui/{button,input,surface,dialog}`),
`lib/` (`config`, `api`, `types`, `username`, `skip`, `session`, `media`,
`signaling`, `webrtc`, `fullscreen`, `call`, `utils`, plus `*.test.ts`),
`public/favicon.svg`, `public/fonts/{InterVar,Fraunces}-{latin,latin-ext}.woff2`,
`server.mjs` (rewritten), `Dockerfile`, `README.md`, `.env.example`.
Frontend, removed from the copy: the Subll landing/dashboard routes and
components, the creator components, the Subll API client and services, Subll
imagery, the Raleway font files, and the 1.4 MB of static Inter TTFs the new
woff2 subsets replace.

Root, new: `README.md`, `scripts/e2e-matchmaking.mjs`, `.baseline/` (checksums
of the untouched source projects).

## 14. Source protection

`documents/subll` and `documents/subllgo` were copied with `rsync` (read-only on
the source) into `frontend/` and `backend/`. Everything since then happened
inside the copies. `node_modules/`, `.git/`, build output, the 25 MB compiled
`subllgo` binary, and the original `.env` (which held real credentials) were
excluded from the copy.

Checksums of both source trees were taken before and after the copy and are
identical, and `git status` in `documents/subll` is clean at revision
`fb9faecd24995ef29b3700b6de896f0e0edf95ad`. The manifests live in `.baseline/`,
so this can be re-verified at any time:

```bash
cd /Users/glo/Documents/subll && git status --porcelain               # expect: empty
shasum -c /Users/glo/Documents/strg/.baseline/subll-checksums.txt     # expect: all OK
shasum -c /Users/glo/Documents/strg/.baseline/subllgo-checksums.txt   # expect: all OK
```

# strangely
