/**
 * The five second Skip window.
 *
 * Every calculation here is derived from the server's match timestamp, so the
 * countdown on screen and the rule the backend enforces are the same rule. The
 * browser clock is only ever used as a difference from a known server time,
 * which is what makes a wrong local clock (or a clock that is deliberately
 * moved) irrelevant.
 */

/**
 * serverOffsetMs is the difference between the server's clock and this
 * browser's, taken from a response's `serverTime` field.
 */
export function serverOffsetMs(serverTime: string, clientNowMs: number): number {
  const parsed = Date.parse(serverTime);
  if (Number.isNaN(parsed)) {
    return 0;
  }
  return parsed - clientNowMs;
}

/**
 * computeRemainingMs is how much of the Skip lock is left, in milliseconds,
 * never below zero.
 */
export function computeRemainingMs(
  matchedAt: string | null | undefined,
  skipLockMs: number,
  clientNowMs: number,
  offsetMs = 0,
): number {
  if (!matchedAt) {
    return 0;
  }

  const matchedAtMs = Date.parse(matchedAt);
  if (Number.isNaN(matchedAtMs)) {
    return 0;
  }

  const elapsed = clientNowMs + offsetMs - matchedAtMs;
  return Math.max(0, skipLockMs - elapsed);
}

/** isSkipLocked reports whether Skip is still refused. */
export function isSkipLocked(remainingMs: number): boolean {
  return remainingMs > 0;
}

/**
 * countdownLabel is the whole second shown next to the Skip button: "5" down
 * to "1". It rounds up, so "5" is shown for the first instant of the call and
 * the label only reaches "1" while a full second is genuinely left.
 */
export function countdownLabel(remainingMs: number): string {
  return String(Math.ceil(Math.max(0, remainingMs) / 1000));
}

/** describeSkipLock is the sentence shown while Skip is unavailable. */
export function describeSkipLock(remainingMs: number): string {
  if (!isSkipLocked(remainingMs)) {
    return "Skip is available.";
  }
  return `Skip available in ${countdownLabel(remainingMs)}…`;
}

/**
 * nextTickDelayMs schedules the next countdown update. While seconds remain it
 * ticks on the second boundary; under a second it polls quickly enough that
 * the button unlocks the instant the server would allow it.
 */
export function nextTickDelayMs(remainingMs: number): number {
  if (remainingMs <= 0) {
    return 250;
  }
  if (remainingMs < 1000) {
    return Math.max(50, remainingMs);
  }

  const remainder = remainingMs % 1000;
  return remainder === 0 ? 1000 : remainder;
}
