import { describe, expect, test } from "bun:test";

import {
  computeRemainingMs,
  countdownLabel,
  describeSkipLock,
  isSkipLocked,
  nextTickDelayMs,
  serverOffsetMs,
} from "./skip";

const MATCHED_AT = "2026-01-01T12:00:00.000Z";
const MATCHED_AT_MS = Date.parse(MATCHED_AT);
const LOCK_MS = 5000;

describe("computeRemainingMs", () => {
  test("starts at the full window", () => {
    expect(computeRemainingMs(MATCHED_AT, LOCK_MS, MATCHED_AT_MS)).toBe(LOCK_MS);
  });

  test("counts down as time passes", () => {
    expect(computeRemainingMs(MATCHED_AT, LOCK_MS, MATCHED_AT_MS + 2000)).toBe(3000);
    expect(computeRemainingMs(MATCHED_AT, LOCK_MS, MATCHED_AT_MS + 4999)).toBe(1);
  });

  test("never goes below zero", () => {
    expect(computeRemainingMs(MATCHED_AT, LOCK_MS, MATCHED_AT_MS + 60_000)).toBe(0);
  });

  test("is zero before a match exists", () => {
    expect(computeRemainingMs(null, LOCK_MS, MATCHED_AT_MS)).toBe(0);
    expect(computeRemainingMs("not a date", LOCK_MS, MATCHED_AT_MS)).toBe(0);
  });

  test("a browser clock that is wrong does not change the window", () => {
    // The browser believes it is two minutes behind the server. The offset
    // from the server's own serverTime repairs it, so the countdown is the
    // same as on a correctly set clock.
    const clientNow = MATCHED_AT_MS + 2000 - 120_000;
    const offset = serverOffsetMs(new Date(MATCHED_AT_MS + 2000).toISOString(), clientNow);

    expect(offset).toBe(120_000);
    expect(computeRemainingMs(MATCHED_AT, LOCK_MS, clientNow, offset)).toBe(3000);
  });

  test("an offset that cannot be parsed is ignored", () => {
    expect(serverOffsetMs("nonsense", MATCHED_AT_MS)).toBe(0);
  });
});

describe("countdownLabel", () => {
  test("shows whole seconds, rounding up so 5 is the first thing seen", () => {
    expect(countdownLabel(5000)).toBe("5");
    expect(countdownLabel(4900)).toBe("5");
    expect(countdownLabel(4001)).toBe("5");
    expect(countdownLabel(4000)).toBe("4");
    expect(countdownLabel(1)).toBe("1");
    expect(countdownLabel(0)).toBe("0");
  });
});

describe("describeSkipLock", () => {
  test("explains the wait, then says it is available", () => {
    expect(describeSkipLock(5000)).toBe("Skip available in 5…");
    expect(describeSkipLock(1000)).toBe("Skip available in 1…");
    expect(describeSkipLock(0)).toBe("Skip is available.");
  });
});

describe("isSkipLocked", () => {
  test("only a positive remaining time locks the button", () => {
    expect(isSkipLocked(1)).toBe(true);
    expect(isSkipLocked(0)).toBe(false);
  });
});

describe("nextTickDelayMs", () => {
  test("ticks on the second boundary while seconds remain", () => {
    expect(nextTickDelayMs(5000)).toBe(1000);
    expect(nextTickDelayMs(4400)).toBe(400);
  });

  test("polls quickly under one second so the unlock is not late", () => {
    expect(nextTickDelayMs(600)).toBe(600);
    expect(nextTickDelayMs(120)).toBe(120);
  });

  test("settles into a slow poll once unlocked", () => {
    expect(nextTickDelayMs(0)).toBe(250);
  });
});
