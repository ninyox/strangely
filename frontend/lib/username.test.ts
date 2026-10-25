import { describe, expect, test } from "bun:test";

import {
  normalizeUsername,
  suggestUsername,
  USERNAME_MAX_LENGTH,
  usernameProblem,
} from "./username";

describe("normalizeUsername", () => {
  test("trims and collapses whitespace", () => {
    expect(normalizeUsername("   Ada   Lovelace  ")).toEqual({ ok: true, value: "Ada Lovelace" });
  });

  test("keeps the punctuation the backend allows", () => {
    expect(normalizeUsername("jo-jo_99")).toEqual({ ok: true, value: "jo-jo_99" });
    expect(normalizeUsername("Ada.L")).toEqual({ ok: true, value: "Ada.L" });
    expect(normalizeUsername("O'Neill")).toEqual({ ok: true, value: "O'Neill" });
  });

  test("accepts non-latin letters", () => {
    expect(normalizeUsername("李雷雷")).toEqual({ ok: true, value: "李雷雷" });
  });

  test("rejects markup and control characters", () => {
    expect(normalizeUsername("<script>alert(1)</script>").ok).toBe(false);
    expect(normalizeUsername("ada\u0007lovelace").ok).toBe(false);
    expect(normalizeUsername("ada\nlovelace").ok).toBe(false);
  });

  test("enforces the length window in runes, not bytes", () => {
    expect(normalizeUsername("ab").ok).toBe(false);
    expect(normalizeUsername("a".repeat(USERNAME_MAX_LENGTH)).ok).toBe(true);
    expect(normalizeUsername("a".repeat(USERNAME_MAX_LENGTH + 1)).ok).toBe(false);
    // 20 accented characters are 20 runes but 40 bytes.
    expect(normalizeUsername("é".repeat(USERNAME_MAX_LENGTH)).ok).toBe(true);
  });

  test("rejects empty input", () => {
    expect(normalizeUsername("    ").ok).toBe(false);
  });
});

describe("usernameProblem", () => {
  test("stays quiet while the visitor is still typing", () => {
    expect(usernameProblem("")).toBeNull();
    expect(usernameProblem("a")).toBeNull();
    expect(usernameProblem("ab")).toBeNull();
  });

  test("reports a real problem once there is one", () => {
    expect(usernameProblem("ab!")).toContain("letters");
    expect(usernameProblem("a".repeat(USERNAME_MAX_LENGTH + 1))).toContain("20");
  });

  test("is quiet for a valid name", () => {
    expect(usernameProblem("Ada Lovelace")).toBeNull();
  });
});

describe("suggestUsername", () => {
  test("suggests something the backend would accept", () => {
    for (let i = 0; i < 20; i += 1) {
      expect(normalizeUsername(suggestUsername()).ok).toBe(true);
    }
  });
});
