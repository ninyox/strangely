/**
 * Username rules.
 *
 * These mirror `utils.NormalizeUsername` in the Go backend on purpose: the form
 * can explain a problem before a request is made, and the server still rejects
 * anything that slips through. A username is a display name for this visit
 * only. It is never an identity and never has to be unique.
 */
export const USERNAME_MIN_LENGTH = 3;
export const USERNAME_MAX_LENGTH = 20;

export type UsernameResult = { ok: true; value: string } | { ok: false; message: string };

const CONTROL_CHARACTERS = /[\u0000-\u001f\u007f-\u009f]/;
const ALLOWED_CHARACTERS = /^[\p{L}\p{N}\s.'_-]+$/u;

/**
 * normalizeUsername trims, collapses runs of whitespace, and enforces the
 * character and length rules. It returns the value the backend would store.
 */
export function normalizeUsername(raw: string): UsernameResult {
  if (CONTROL_CHARACTERS.test(raw)) {
    return { ok: false, message: "Use letters, numbers, spaces, and . _ - only." };
  }

  const value = raw.trim().split(/\s+/).filter(Boolean).join(" ");

  if (value === "") {
    return { ok: false, message: "Please choose a name." };
  }
  if (!ALLOWED_CHARACTERS.test(value)) {
    return { ok: false, message: "Use letters, numbers, spaces, and . _ - only." };
  }

  const length = [...value].length;
  if (length < USERNAME_MIN_LENGTH) {
    return { ok: false, message: `Names need at least ${USERNAME_MIN_LENGTH} characters.` };
  }
  if (length > USERNAME_MAX_LENGTH) {
    return { ok: false, message: `Names can be at most ${USERNAME_MAX_LENGTH} characters.` };
  }

  return { ok: true, value };
}

/**
 * usernameProblem returns the message to show under the field, or null when
 * the value is fine. It stays quiet while the visitor is still typing
 * something shorter than the minimum.
 */
export function usernameProblem(raw: string): string | null {
  if (raw.trim() === "") {
    return null;
  }

  const result = normalizeUsername(raw);
  if (result.ok) {
    return null;
  }

  // Do not shout "too short" at somebody who has typed two characters.
  if ([...raw.trim()].length < USERNAME_MIN_LENGTH && result.message.startsWith("Names need")) {
    return null;
  }

  return result.message;
}

/** A friendly default name for a first visit, so nobody is stuck on the form. */
export function suggestUsername(): string {
  const adjectives = ["Quiet", "Bright", "Amber", "Kind", "Swift", "Curious", "Steady", "Sunny"];
  const nouns = ["Otter", "Finch", "Cedar", "Comet", "Harbor", "Lark", "Willow", "Falcon"];

  const pick = <T,>(list: T[]): T => list[Math.floor(Math.random() * list.length)];

  return `${pick(adjectives)}${pick(nouns)}${Math.floor(10 + Math.random() * 89)}`;
}
