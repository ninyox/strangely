# Agent Build Specification — Strangerly

## 1. Product Overview

Build **Strangerly**, a clean, professional, mobile-first random video-chat website where visitors can meet strangers for spontaneous one-to-one conversations.

**Product tagline:** *One click away from someone new.*

The experience should feel modern, safe-minded, fast, and intentionally designed—not like a dated Omegle clone and not like a generic SaaS dashboard.

### Core user journey

1. A visitor opens the site without creating an account.
2. They choose a username for the current visit.
3. They allow camera and microphone access when prompted.
4. They press **Start matching**.
5. The app queues them and matches them with another available visitor.
6. Both people connect to a one-to-one live video chat.
7. The **Skip** action is unavailable for the first **5 seconds after the match is established**. Show a small, clear countdown so this behavior is understandable.
8. After five seconds, either participant may skip/end the chat and return to matching.
9. A visitor can stop matching or leave at any time. The five-second restriction applies to the Skip action only; it must never trap a user in a conversation or prevent them from closing the tab, stopping their camera, or ending the call for safety.

No registration, email, password, OAuth, or account system. Use a browser cookie to identify a temporary visit/session, with a server-generated opaque session ID. Do not treat a username or cookie as proof of identity.

## 2. Mandatory Workspace and Source Protection

The user has an existing React project at `documents/subll` and an existing Go project at `documents/subllgo`.

**Absolute requirements:**
- Copy `documents/subll` into this folder`.
- Copy `documents/subllgo` into a new backend folder named `strgback`, for example.
- **Never modify, rename, delete, move, format, install dependencies into, or write generated files into the original `documents/subll` or `documents/subllgo` directories.**
- All edits, dependency installs, build artifacts, generated files, and configuration changes must happen only in the new `strangerly` copy.
- Do not use symlinks to the original project directories.
- Verify the exact source and destination paths before copying. If the paths cannot be found, inspect the current workspace and report the issue rather than guessing or overwriting anything.
- Never run destructive commands against the source directories. Avoid commands such as `rm -rf`, `git clean`, or formatters with source paths that could touch either original directory.
- Before and after copying, verify the originals remain unchanged. Prefer recording checksums or a read-only git status/diff if available.
- Preserve useful React project structure, tooling, and conventions, but remove/replace Subll-specific UI and product logic **only inside the copied `strangerly` frontend**.
- Preserve useful Go project structure and dependencies in the copied backend, but replace Subll-specific business logic **only inside `strgback`**.
- Do not assume the React project and Go project are in the current working directory. Find the user’s `documents` directory and confirm the exact paths first.

Expected layout (adapt only if the copied projects require a different safe structure):

```text
strangerly/
  frontend/        # copy of documents/subll
  backend/         # copy of documents/subllgo, adapted for Strangerly
  README.md
  prompt.md        # this specification, copied into the project
```

If the user expects the React project to be copied directly into `strangerly/` rather than `strangerly/frontend/`, inspect existing tooling and choose a sensible layout while keeping the frontend and backend isolated. Do not put the backend inside the React app's source tree.

## 3. Design Direction

### Brand
- Name: **Strangerly**
- Tagline: **One click away from someone new.**
- Voice: warm, direct, confident, contemporary, respectful.
- Main background: clean white / warm off-white.
- Accent: vivid orange-red (for example `#F04438` or a nearby accessible shade).
- Text: near-black / charcoal.
- Supporting surfaces: very light warm gray.
- Borders: subtle, restrained, low-contrast.
- Use color intentionally; orange-red is the key action accent, not a background for every component.

### Visual language
- Premium, editorial minimalism.
- Strong typography and generous whitespace.
- Large, clear video area.
- Rounded corners with restrained radii, subtle borders, and soft shadows only where useful.
- Small, purposeful motion for matching, connection, and state changes.
- Clear focus states and accessible contrast.
- Responsive from small phones to wide desktop screens.
- No cluttered dashboard, no unnecessary gradients, no excessive glassmorphism, no emoji-heavy interface, no generic template look.
- Do not use fabricated testimonials, fake user counts, fake online activity, or unsupported safety claims.

### Suggested landing layout
1. Minimal top bar: Strangerly wordmark, short descriptor, and a compact safety/help link.
2. Main hero with a concise headline such as **“Meet someone new.”** and one supporting sentence.
3. Main interaction card with:
   - Username input
   - Camera preview / permission state
   - Primary **Start matching** button
   - Brief privacy and conduct note
4. Small, tasteful explanatory row: “Choose a name · Match instantly · Talk one-on-one.”
5. Concise safety guidance and report/block affordances where relevant.
6. Footer with Privacy, Community Guidelines, and About links. If these pages are included, write real, honest content rather than placeholder legal promises.

### Main chat screen
- Make the remote participant’s video the visual focus.
- Show the local camera preview as a smaller picture-in-picture tile, movable only if implemented robustly.
- Show the other participant’s chosen username without implying identity verification.
- Controls should be obvious and accessible: **Mute**, **Camera**, **Skip**, **Stop**.
- During matching, show a calm animated searching state and a **Cancel** action.
- When waiting for camera permission, device availability, or a network connection, explain the actual state clearly.
- On match, show a short connecting state, then the connected video chat.
- While Skip is locked, visibly communicate **“Skip available in 5…4…3…2…1”** based on server-authoritative elapsed time, not a client-only timer.
- Once the restriction expires, make Skip available. Stopping the call and emergency/safety actions remain available at all times.
- If a partner disconnects, show a clean end state with **Find someone new** and **Stop**.

## 4. Frontend Requirements

Use the copied React project as the starting point. Inspect its package manager, build tool, routing, styling system, and existing conventions before editing.

- Keep the implementation in the existing stack where practical; do not rewrite the project unnecessarily.
- Use TypeScript if the copied project already uses it, or introduce it only if that can be done without a costly rewrite.
- Use reusable, typed components and clear state management.
- Keep loading, empty, permission-denied, device-error, network-error, matching, connecting, connected, disconnected, and reconnecting states distinct.
- Avoid dead buttons and fake interactions.
- Validate and normalize usernames on both frontend and backend. Trim whitespace, set a reasonable length limit (for example 3–20 characters), reject control characters, and escape user-supplied text by rendering it as text rather than HTML.
- A username is only a display name. Do not require it to be unique globally.
- Do not put secrets, privileged configuration, or private keys in the frontend.
- Do not store video, audio, or chat history. There is no text-chat feature in the initial release unless explicitly required later.
- Persist only the minimum session information needed for a temporary visit. Never put camera/microphone media or sensitive personal data into cookies or localStorage.
- Use browser-native `getUserMedia` for camera and microphone access, with explicit user permission and clear handling when permission is denied.
- Release media tracks and close peer connections when a user stops, skips, disconnects, or leaves.
- The page must work well on mobile browsers; provide an understandable fallback if camera access or autoplay is restricted.
- Use a clean favicon, metadata, page title, and responsive layout.

## 5. Go Backend Requirements

Build the backend in Go, in the new copied backend folder only. Inspect the copied Go project's framework and architecture first; reuse appropriate dependencies and patterns, but remove Subll-specific routes and business logic in the copy.

### Core backend responsibilities
- Create and manage anonymous temporary sessions.
- Maintain a matchmaking queue.
- Match two distinct eligible sessions atomically.
- Coordinate WebRTC signaling between the two matched clients.
- Track match lifecycle and disconnects.
- Enforce the five-second Skip lock using server timestamps.
- Handle explicit stop, skip, partner disconnect, session expiry, and stale queue entries.
- Provide health/readiness endpoints suitable for deployment.
- Use structured logs without logging sensitive media, session secrets, or unnecessary personal data.

### Suggested API design
Adjust names to match the existing code conventions, but implement equivalent functionality:

- `POST /api/session` — create or resume a temporary anonymous session; set an opaque, random, `HttpOnly`, `SameSite=Lax` cookie. Use `Secure` in HTTPS production. Do not trust client-provided session IDs.
- `POST /api/matchmaking/join` — enqueue a session for matching.
- `POST /api/matchmaking/cancel` — remove a session from the queue if still waiting.
- `GET /api/matchmaking/status` — return the current state and match ID for the session.
- `POST /api/matches/{id}/skip` — request a skip, returning a clear conflict/too-early response if the five-second lock has not expired.
- `POST /api/matches/{id}/stop` — end the current call immediately.
- `POST /api/matches/{id}/report` — submit a minimal report against the current match; rate-limit and avoid storing unnecessary identifying information.
- `GET /api/signaling` — authenticated-by-session WebSocket signaling endpoint, or an equivalent WebSocket route, for exchanging WebRTC offer, answer, and ICE candidate messages.
- `GET /healthz` — process health.
- `GET /readyz` — readiness, including required dependencies if any.

If a WebSocket is used, validate the Origin against the configured frontend origin, authenticate using the temporary session cookie, limit message sizes, validate message types and match membership, and enforce read/write deadlines and heartbeat timeouts.

### Session and queue design
- Use cryptographically secure random session identifiers.
- Store only a hash of the session token server-side where practical.
- Add TTL/expiry for sessions, queue entries, and matches.
- A session may only occupy one queue entry or one active match at a time.
- Do not match a session with itself.
- Prevent duplicate concurrent join requests and duplicate active matches with synchronization or transactional operations.
- Clean up stale sessions and abandoned queue entries.
- Handle a client that closes the tab or loses connectivity.
- In a single-process initial implementation, use safe in-memory state with a clear interface so Redis can be added later. If the copied project already has a suitable Redis integration, reuse it only if the configuration and reliability benefits are justified. Do not add unnecessary infrastructure.
- Make the matching process fair (FIFO among eligible waiting sessions) and resistant to duplicate requests.
- Return clear state values such as `idle`, `queued`, `connecting`, `matched`, and `ended`.

## 6. WebRTC and Live Video

Use **WebRTC** for peer-to-peer audio/video. The Go server coordinates signaling; it should not relay raw video or audio.

- Exchange SDP offers/answers and ICE candidates only between the two participants in the same active match.
- Validate every signaling message against the authenticated session and current match.
- Configure ICE servers via environment variables.
- Document that a STUN server alone does not guarantee connectivity across every NAT/firewall. Support optional TURN configuration through environment variables for production reliability.
- Never hardcode private TURN credentials in the frontend bundle. For production, use short-lived TURN credentials generated server-side or a trusted managed provider.
- Request camera/microphone access only after a clear user action.
- Display local and remote connection states, and handle `failed`, `disconnected`, `closed`, and track-ended cases.
- If no remote stream is available, show a clear connecting state rather than a broken video element.
- Do not claim that all calls are end-to-end secure beyond the guarantees actually provided by WebRTC and the chosen configuration.
- Never record calls or capture media for analytics.

## 7. Five-Second Skip Rule

This rule must be enforced by the backend, not just by disabling a button in React.

- Record `matchedAt` on the server when the two sessions are paired.
- A Skip request before `matchedAt + 5 seconds` must be rejected with an explicit response containing the remaining time.
- Calculate remaining time from the server's match timestamp; client timers are presentation only.
- Once five seconds have elapsed, either participant can skip. Define the semantics clearly: a successful skip ends the current match for both participants, cleans up the peer connection/signaling state, and returns the requesting user to matchmaking only if they explicitly choose to continue or if the UI clearly asks whether to find someone new.
- The other participant must be notified promptly and shown a **“Your partner left”** state.
- The Skip restriction must never disable **Stop**, camera-off, microphone mute, report/block controls, browser navigation, or closing the page.
- Handle simultaneous skip/stop requests idempotently.
- Test server/client clock skew and delayed status updates.

## 8. Safety, Privacy, and Abuse Prevention

Random video chat can expose users to harassment, nudity, scams, and unwanted contact. Build safety affordances into the first version.

- Display a brief community-guidelines/conduct note before a visitor starts.
- Provide a report action that is always available during a call. Reporting must not be blocked by the five-second Skip lock.
- Provide an immediate stop/end-call action.
- Provide an optional local block/avoid list if it can be implemented meaningfully without identity accounts; explain that a local block list may not persist across devices and cannot reliably identify someone who changes sessions.
- Apply rate limits to session creation, matchmaking, signaling, and reporting.
- Validate all input, limit payload sizes, reject malformed messages, and prevent cross-session signaling.
- Do not expose session tokens in URLs, logs, analytics, or error messages.
- Configure strict CORS to the known frontend origin; do not use wildcard credentials.
- Set appropriate security headers and a restrictive Content Security Policy compatible with WebRTC.
- Avoid storing IP addresses longer than operationally necessary. If infrastructure logs retain IPs, document this honestly.
- Provide real privacy and community-guideline copy; do not claim that the service is moderated, anonymous in an absolute sense, or guaranteed safe unless that is true.
- Consider an age gate and clear “adults only” positioning if the product is intended for adults; do not collect identity documents. Do not imply the age gate verifies age.
- The initial release should not include public profiles, friend lists, persistent direct messaging, user discovery by location, or media recording.

## 9. Configuration and Deployment

- Keep secrets and deployment settings in environment variables.
- Add a `.env.example` containing placeholder values only; never commit real credentials.
- Include settings for:
  - frontend origin / allowed WebSocket Origin
  - HTTP listen address and port
  - cookie secure mode and session TTL
  - allowed CORS origin
  - STUN URL(s)
  - optional TURN URL, username, and short-lived credential mechanism
  - rate-limit settings
  - log level
- Keep local development straightforward: document commands to install dependencies, run the React frontend, and run the Go backend.
- Configure the frontend to use a single documented API base URL.
- Add a deployment guide for serving the frontend and proxying `/api` and WebSocket traffic to Go. Include correct WebSocket upgrade handling for the chosen reverse proxy.
- Do not assume a database is required for the first single-process version. Clearly document that in-memory matchmaking state is lost on backend restart and does not support multiple backend replicas without shared state.
- Provide a practical production path for adding Redis and TURN without making them mandatory for the initial local prototype.

## 10. Testing and Acceptance Criteria

Implement tests where practical and manually verify the full journey.

### Backend tests
- Session creation uses secure opaque IDs and appropriate cookies.
- Invalid usernames are rejected or normalized consistently.
- Two different waiting sessions are matched exactly once.
- A session cannot match itself or enter the queue multiple times.
- A session cannot join matchmaking while already in an active match.
- Skip requests before five seconds are rejected.
- Skip requests after five seconds end the match idempotently.
- Stop always works, including during the five-second lock.
- Partner disconnects and stale queue entries are cleaned up.
- Signaling messages cannot be sent to sessions outside the caller's match.
- Invalid Origin, oversized messages, expired sessions, and rate-limited requests are handled safely.

### Frontend acceptance
- A visitor can set a username without registering.
- Camera/microphone permission is requested only after an intentional action.
- Camera permission denial and missing-device errors are explained clearly.
- Start matching, cancel matching, match, connect, skip, stop, and partner-left flows work.
- Skip is visibly locked for five seconds and backend-enforced.
- Mute and camera toggles work and accurately reflect their state.
- Local media tracks and peer connections are cleaned up when ending a call.
- Responsive layout works on narrow mobile screens and desktop.
- Keyboard navigation, visible focus indicators, labels, and contrast are usable.
- No primary control is a fake button or a dead end.
- Reloading or losing connection does not leave a session permanently stuck in the queue.

### Manual two-browser test
Use two separate browser profiles or devices:
1. Open Strangerly in both.
2. Set different usernames and allow camera/microphone.
3. Start matching in both.
4. Verify they match and can see/hear each other.
5. Verify Skip is rejected for the first five seconds.
6. Verify Stop and report remain available during those five seconds.
7. After five seconds, skip from one side and confirm the other side sees the disconnect.
8. Confirm both users can start a fresh match.
9. Repeat with one side closing its tab and with one side denying camera access.

If camera testing is not possible in the current environment, state this explicitly. Do not claim that real peer-to-peer video has been verified unless it has been tested.

## 11. Deliverables

Complete the working project, not just mockups.

1. Copied React frontend adapted to Strangerly.
2. Copied Go backend adapted to Strangerly.
3. Working anonymous session cookie flow.
4. Working matchmaking queue and match lifecycle.
5. WebRTC signaling and one-to-one video chat.
6. Server-enforced five-second Skip lock.
7. Safety/report/stop affordances.
8. `.env.example` files with placeholders.
9. `README.md` with architecture, setup, local development, tests, known limitations, and deployment.
10. A concise list of changed files and the commands needed to run the app.
11. Any remaining limitations stated honestly, especially around TURN requirements, moderation, persistence, and production scaling.

## 12. Execution Plan

Work in this order:

1. Inspect the workspace and locate `documents/subll` and `documents/subllgo`.
2. Verify source paths and establish a safe copy strategy.
3. Copy both source projects into the new `strangerly` workspace. Never edit the originals.
4. Inspect frontend/backend architecture and identify which parts can safely be reused.
5. Write a short implementation plan and then execute it; do not stop after planning.
6. Build the frontend design and all UI states.
7. Implement anonymous sessions, queue management, match lifecycle, and server-side Skip enforcement.
8. Implement WebRTC signaling and media cleanup.
9. Add validation, rate limits, safety controls, tests, and documentation.
10. Run frontend lint/build and backend tests/build. Fix issues in the copied project only.
11. Perform the two-browser test if the environment supports it.
12. Verify that the original `documents/subll` and `documents/subllgo` directories were not changed.
13. Summarize what works, exact run commands, and any unverified behavior.

**Do not stop at a pretty static UI.** The objective is a functional, tested first version with a strong visual identity and a real matching/video flow. When a feature cannot be completed or verified, be transparent rather than simulating success.
