# Strangerly frontend

A React Router (SPA) + TypeScript + Tailwind v4 client for Strangerly, the
random one-to-one video chat.

## Commands

```bash
bun install        # install dependencies
bun run dev        # development server on http://localhost:5173
bun run typecheck  # react-router typegen + tsc
bun run test       # bun test (pure logic: username rules, skip window maths)
bun run build      # production build into build/client
bun run start      # serve build/client with server.mjs on http://localhost:3000
```

The API base URL comes from a single setting, `VITE_API_BASE_URL` (see
`.env.example`). It defaults to `http://localhost:8080`, and the signaling
WebSocket URL and the Content Security Policy `connect-src` are both derived
from it.

## Layout

```text
app/
  app.css                  the design system: fonts, tokens, utilities, motion
  root.tsx                 HTML shell + Session/Media/Call providers
  routes.ts                route table
  routes/
    home.tsx               landing: hero, how it works, safety, closing CTA
    chat.tsx               every call state in one dark stage
    privacy.tsx            real privacy copy
    guidelines.tsx         real community guidelines
    about.tsx              what this is and is not
    not-found.tsx
  components/
    wordmark.tsx           the two-ring mark and the wordmark
    hero.tsx               the asymmetric hero composition
    live-panel.tsx         the product as a panel: name, camera, start
    camera-preview.tsx     broadcast monitor chrome around the preview
    how-it-works.tsx       the dark band: numbered steps
    stage-preview.tsx      the call stage in its waiting state
    safety-section.tsx     the clay band
    closing-cta.tsx        the signal band
    call-stage.tsx         remote video, local PiP, state strip
    call-controls.tsx      the sticky control bar
    countdown-ring.tsx     the Skip lock, drawn
    state-panels.tsx       searching, preparing, failed, ended
    ui/                    button, input, surface, dialog primitives
lib/
  api.ts                   typed API client (one ApiError shape)
  call.tsx                 the call state machine
  config.ts                the single API base URL
  media.tsx                getUserMedia with honest failure states
  session.tsx              the anonymous visit
  signaling.ts             WebSocket client with reconnect
  skip.ts                  server-authoritative Skip countdown maths
  types.ts                 payload types shared with the Go backend
  username.ts              display-name rules, mirrored from the backend
  webrtc.ts                RTCPeerConnection wrapper
```

The full project documentation — art direction, architecture, deployment, and
known limitations — lives in the repository root `README.md`.
