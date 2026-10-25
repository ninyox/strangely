#!/usr/bin/env node
/**
 * Strangerly end-to-end check.
 *
 * Two anonymous visitors are simulated against a running backend: separate
 * cookie jars, two signaling sockets, a real match, a real SDP/ICE exchange,
 * and the five second Skip rule. It is the scripted half of the two-browser
 * test; it cannot verify that a camera produced pixels, only that every step
 * of the protocol the browsers depend on behaves.
 *
 * Usage:
 *   node scripts/e2e-matchmaking.mjs                # against http://127.0.0.1:8080
 *   STRG_API_BASE=http://127.0.0.1:9090 node scripts/e2e-matchmaking.mjs
 *
 * Exits non-zero if any check fails.
 */

const API_BASE = (process.env.STRG_API_BASE ?? "http://127.0.0.1:8080").replace(/\/+$/, "");
const WS_BASE = API_BASE.replace(/^http/, "ws");

// The signaling socket checks the Origin against the server's allowlist, so
// the simulated browser has to announce an allowed origin — exactly as a real
// browser would. Set STRG_ORIGIN if the backend allows something else.
const ORIGIN = process.env.STRG_ORIGIN ?? "http://localhost:5173";

const results = [];

function check(name, ok, detail = "") {
  results.push({ name, ok });
  const mark = ok ? "\u001b[32mPASS\u001b[0m" : "\u001b[31mFAIL\u001b[0m";
  console.log(`${mark}  ${name}${detail ? ` — ${detail}` : ""}`);
}

function section(title) {
  console.log(`\n\u001b[1m${title}\u001b[0m`);
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

/** One simulated browser: its own cookie jar and its own signaling socket. */
class Visitor {
  constructor(label) {
    this.label = label;
    this.cookie = null;
    this.socket = null;
    this.messages = [];
    this.waiters = [];
  }

  async request(method, path, body) {
    const headers = { "Content-Type": "application/json" };
    if (this.cookie) {
      headers.Cookie = this.cookie;
    }

    const response = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });

    for (const raw of response.headers.getSetCookie?.() ?? []) {
      const pair = raw.split(";")[0];
      if (pair.startsWith("strg_session=")) {
        this.cookie = pair;
      }
    }

    let payload = null;
    try {
      payload = await response.json();
    } catch {
      payload = null;
    }

    return { status: response.status, body: payload };
  }

  createSession(username) {
    return this.request("POST", "/api/session", username ? { username } : {});
  }

  join(username) {
    return this.request("POST", "/api/matchmaking/join", username ? { username } : {});
  }

  status() {
    return this.request("GET", "/api/matchmaking/status");
  }

  async openSocket() {
    if (!this.cookie) {
      throw new Error(`${this.label} has no session cookie`);
    }

    const socket = new WebSocket(`${WS_BASE}/api/signaling`, {
      headers: { Cookie: this.cookie, Origin: ORIGIN },
    });
    this.socket = socket;

    await new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(`${this.label}: signaling socket timed out`)), 5000);
      socket.onopen = () => {
        clearTimeout(timer);
        resolve();
      };
      socket.onclose = () => {
        clearTimeout(timer);
        reject(new Error(`${this.label}: signaling socket closed before opening`));
      };
    });

    socket.onmessage = (event) => {
      let message;
      try {
        message = JSON.parse(event.data);
      } catch {
        return;
      }

      // A waiting caller consumes the message; anything else is buffered so a
      // later waitFor (or receivedWithin) can still see it.
      const waiterIndex = this.waiters.findIndex((waiter) => waiter.type === message.type);
      if (waiterIndex !== -1) {
        const [waiter] = this.waiters.splice(waiterIndex, 1);
        clearTimeout(waiter.timer);
        waiter.resolve(message);
        return;
      }

      this.messages.push(message);
    };

    return this;
  }

  send(type, matchId, payload) {
    this.socket.send(JSON.stringify({ type, matchId, payload }));
  }

  waitFor(type, timeoutMs = 5000) {
    const buffered = this.messages.findIndex((message) => message.type === type);
    if (buffered !== -1) {
      const [message] = this.messages.splice(buffered, 1);
      return Promise.resolve(message);
    }

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.waiters = this.waiters.filter((waiter) => waiter.timer !== timer);
        reject(new Error(`${this.label}: never received a "${type}" message`));
      }, timeoutMs);
      this.waiters.push({ type, resolve, timer });
    });
  }

  /** Wait a moment and report whether a message of this type arrived. */
  async receivedWithin(type, ms) {
    await sleep(ms);
    const index = this.messages.findIndex((message) => message.type === type);
    if (index === -1) {
      return false;
    }
    this.messages.splice(index, 1);
    return true;
  }

  close() {
    this.socket?.close();
  }
}

async function main() {
  console.log(`Strangerly end-to-end check against ${API_BASE}`);

  /* ------------------------------------------------------------------ */
  section("1. Health and public configuration");

  const health = await fetch(`${API_BASE}/healthz`).then((response) => response.json());
  check("GET /healthz reports ok", health?.data?.status === "ok");

  const readyResponse = await fetch(`${API_BASE}/readyz`);
  const ready = await readyResponse.json();
  check("GET /readyz reports ready", readyResponse.status === 200 && ready?.data?.status === "ready");

  const config = await fetch(`${API_BASE}/api/config`).then((response) => response.json());
  const skipLockSeconds = config?.data?.skipLockSeconds;
  check(
    "GET /api/config publishes the Skip window",
    typeof skipLockSeconds === "number" && skipLockSeconds > 0,
    `skipLockSeconds=${skipLockSeconds}`,
  );
  check(
    "GET /api/config publishes ICE servers",
    Array.isArray(config?.data?.iceServers) && config.data.iceServers.length > 0,
  );
  check(
    "GET /api/config publishes report reasons",
    Array.isArray(config?.data?.reportReasons) && config.data.reportReasons.length > 0,
  );

  /* ------------------------------------------------------------------ */
  section("2. Anonymous sessions");

  const ada = new Visitor("Ada");
  const grace = new Visitor("Grace");

  const adaSession = await ada.createSession("Ada");
  const graceSession = await grace.createSession("Grace");
  check(
    "POST /api/session creates a session",
    adaSession.status === 200 && Boolean(adaSession.body?.data?.sessionId),
  );
  check("the session cookie is set", Boolean(ada.cookie?.startsWith("strg_session=")));
  check(
    "two visitors get different sessions",
    adaSession.body?.data?.sessionId !== graceSession.body?.data?.sessionId,
  );

  const resumed = await ada.createSession("Ada");
  check(
    "POST /api/session resumes the existing visit",
    resumed.body?.data?.sessionId === adaSession.body?.data?.sessionId,
  );

  const badName = await ada.createSession("<script>alert(1)</script>");
  check(
    "an invalid display name is rejected",
    badName.status === 400 && badName.body?.code === "invalid_username",
    `status=${badName.status} code=${badName.body?.code}`,
  );

  const anonymous = await fetch(`${API_BASE}/api/matchmaking/status`);
  check("protected routes reject a visitor with no session", anonymous.status === 401);

  /* ------------------------------------------------------------------ */
  section("3. Queue, match, and signaling");

  await ada.openSocket();
  await grace.openSocket();
  check("both signaling sockets open with the session cookie", true);

  const hello = await ada.waitFor("hello");
  check(
    "the socket greets the client with its state",
    hello?.data?.sessionId === adaSession.body.data.sessionId,
  );

  const adaJoin = await ada.join("Ada");
  check(
    "the first visitor waits in the queue",
    adaJoin.status === 200 && adaJoin.body?.data?.state === "queued",
  );

  const graceJoin = await grace.join("Grace");
  const matchId = graceJoin.body?.data?.matchId;
  check(
    "the second visitor is matched immediately",
    graceJoin.status === 200 && Boolean(matchId),
    `state=${graceJoin.body?.data?.state}`,
  );
  check("the second visitor sees the first visitor's name", graceJoin.body?.data?.partnerUsername === "Ada");
  check("the second visitor is not the initiator", graceJoin.body?.data?.initiator === false);

  const adaMatched = await ada.waitFor("matched");
  const graceMatched = await grace.waitFor("matched");
  check(
    "both sides receive the matched event",
    adaMatched?.data?.matchId === matchId && graceMatched?.data?.matchId === matchId,
  );
  check("the first visitor is the initiator", adaMatched?.data?.initiator === true);
  check("exactly one side is the initiator", adaMatched?.data?.initiator !== graceMatched?.data?.initiator);

  const adaStatus = await ada.status();
  check(
    "status agrees on the match",
    adaStatus.body?.data?.matchId === matchId && adaStatus.body?.data?.partnerUsername === "Grace",
  );
  check(
    "status reports the server-side Skip window",
    adaStatus.body?.data?.remainingMs > 0 &&
      adaStatus.body?.data?.skipLockMs === Math.round(skipLockSeconds * 1000),
    `remainingMs=${adaStatus.body?.data?.remainingMs}`,
  );

  // A real SDP/ICE exchange, in the exact shape the browser sends.
  ada.send("offer", matchId, { type: "offer", sdp: "v=0\r\no=- 1 1 IN IP4 127.0.0.1\r\ns=-\r\n" });
  const relayedOffer = await grace.waitFor("offer");
  check("the offer reaches the partner", relayedOffer?.data?.payload?.sdp?.includes("v=0"));

  grace.send("answer", matchId, { type: "answer", sdp: "v=0\r\no=- 2 2 IN IP4 127.0.0.1\r\ns=-\r\n" });
  const relayedAnswer = await ada.waitFor("answer");
  check("the answer reaches the offerer", relayedAnswer?.data?.payload?.sdp?.includes("o=- 2"));

  ada.send("ice", matchId, { candidate: "candidate:1 1 udp 2113937151 192.0.2.1 5000 typ host" });
  const relayedIce = await grace.waitFor("ice");
  check("ICE candidates are relayed", relayedIce?.data?.payload?.candidate?.startsWith("candidate:1"));

  // Both sides report that their peer connection is up. Whichever socket the
  // server processes second triggers the notification to the other one, so the
  // check accepts either order rather than assuming one.
  ada.send("ready", matchId);
  grace.send("ready", matchId);

  const notified = await Promise.race([
    ada
      .waitFor("partner_ready", 4000)
      .then(() => "Ada")
      .catch(() => null),
    grace
      .waitFor("partner_ready", 4000)
      .then(() => "Grace")
      .catch(() => null),
  ]);
  check("the side that was already ready is notified", Boolean(notified), notified ? `notified: ${notified}` : "neither side was notified");

  const matchedState = await ada.status();
  check(
    "both sides being ready moves the state to matched",
    matchedState.body?.data?.state === "matched",
    `state=${matchedState.body?.data?.state}`,
  );


  /* ------------------------------------------------------------------ */
  section("4. The five second Skip rule");

  const earlySkip = await ada.request("POST", `/api/matches/${matchId}/skip`);
  check(
    "Skip inside the window is refused",
    earlySkip.status === 409 && earlySkip.body?.code === "skip_locked",
    `status=${earlySkip.status} code=${earlySkip.body?.code}`,
  );

  const remainingMs = earlySkip.body?.data?.remainingMs;
  check(
    "the refusal carries the remaining time",
    typeof remainingMs === "number" &&
      remainingMs > 0 &&
      remainingMs <= Math.round(skipLockSeconds * 1000),
    `remainingMs=${remainingMs}`,
  );
  check("the refusal did not end the call", (await ada.status()).body?.data?.matchId === matchId);
  check(
    "the partner was not told about the refused Skip",
    !(await grace.receivedWithin("partner_left", 400)),
  );

  /* ------------------------------------------------------------------ */
  section("5. Safety actions are never blocked");

  const duringLock = await ada.request("POST", `/api/matches/${matchId}/report`, {
    reason: "harassment_or_hate",
    note: "e2e check",
  });
  check(
    "reporting works inside the Skip window",
    duringLock.status === 200 && Boolean(duringLock.body?.data?.reportId),
  );

  const badReason = await ada.request("POST", `/api/matches/${matchId}/report`, { reason: "not-a-reason" });
  check("an unknown report reason is rejected", badReason.status === 400 && badReason.body?.code === "invalid_reason");

  const outsider = new Visitor("Outsider");
  await outsider.createSession("Outsider");
  const outsiderSkip = await outsider.request("POST", `/api/matches/${matchId}/skip`);
  check(
    "another session cannot end somebody else's call",
    outsiderSkip.status === 404 && outsiderSkip.body?.code === "match_not_found",
  );

  await outsider.openSocket();
  outsider.send("offer", matchId, { type: "offer", sdp: "v=0\r\nspoofed" });
  const outsiderError = await outsider.waitFor("error");
  check("another session cannot signal into the call", outsiderError?.data?.code === "not_in_match");
  check("the spoofed offer never reached the pair", !(await grace.receivedWithin("offer", 400)));

  const stopDuringLock = await ada.request("POST", `/api/matches/${matchId}/stop`);
  check(
    "Stop works inside the Skip window",
    stopDuringLock.status === 200 && stopDuringLock.body?.data?.reason === "stopped",
  );

  const graceLeft = await grace.waitFor("partner_left");
  check("the partner is told the call ended", graceLeft?.data?.reason === "stopped");

  const graceAfterStop = await grace.status();
  check(
    "the partner is free to match again",
    graceAfterStop.body?.data?.state === "ended" && !graceAfterStop.body?.data?.matchId,
  );

  const repeatedStop = await ada.request("POST", `/api/matches/${matchId}/stop`);
  check("Stop is idempotent", repeatedStop.status === 200 && repeatedStop.body?.data?.alreadyEnded === true);

  /* ------------------------------------------------------------------ */
  section("6. A fresh match, then a Skip after the window");

  const graceRejoin = await grace.join("Grace");
  check("the partner can rejoin the queue", graceRejoin.status === 200 && graceRejoin.body?.data?.state === "queued");

  const adaRejoin = await ada.join("Ada");
  const secondMatchId = adaRejoin.body?.data?.matchId;
  check("a new match is created", Boolean(secondMatchId) && secondMatchId !== matchId);
  await ada.waitFor("matched");
  await grace.waitFor("matched");

  console.log(`   waiting ${skipLockSeconds}s for the Skip window to expire…`);
  await sleep(skipLockSeconds * 1000 + 300);

  const skip = await ada.request("POST", `/api/matches/${secondMatchId}/skip`);
  check(
    "Skip is allowed after the window",
    skip.status === 200 && skip.body?.data?.reason === "skipped",
    `status=${skip.status}`,
  );

  const graceSkipped = await grace.waitFor("partner_left");
  check("the partner is told about the Skip", graceSkipped?.data?.reason === "skipped");

  const graceFinal = await grace.status();
  check(
    "the skipped call is over for both",
    graceFinal.body?.data?.state === "ended" && graceFinal.body?.data?.lastEndedReason === "skipped",
  );

  const adaFinal = await ada.status();
  check(
    "the caller is back to the ended state",
    adaFinal.body?.data?.state === "ended" && !adaFinal.body?.data?.matchId,
  );

  ada.close();
  grace.close();
  outsider.close();
}

main()
  .catch((error) => {
    console.error(`\n\u001b[31mThe check stopped early:\u001b[0m ${error.message}`);
    results.push({ name: "the check ran to completion", ok: false });
  })
  .finally(() => {
    const failed = results.filter((result) => !result.ok);
    console.log(
      `\n${results.length - failed.length}/${results.length} checks passed` +
        (failed.length ? `; failed: ${failed.map((f) => f.name).join(", ")}` : ""),
    );
    process.exit(failed.length ? 1 : 0);
  });

