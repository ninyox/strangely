import { describe, expect, test } from "bun:test";

import { SignalingClient, type SignalingHandlers } from "./signaling";

/**
 * A socket stand-in: waitForOpen only ever reads `readyState` and close() calls
 * `close`, so nothing else has to exist for these checks.
 */
type SocketStub = { readyState: number; close: () => void };

function newClient(): SignalingClient {
  const handlers: SignalingHandlers = {
    onEvent: () => {},
    onOpen: () => {},
    onClose: () => {},
  };

  return new SignalingClient("ws://example.invalid/api/signaling", handlers);
}

function stubSocket(client: SignalingClient, readyState: number): SocketStub {
  const socket: SocketStub = { readyState, close: () => {} };
  (client as unknown as { socket: SocketStub | null }).socket = socket;
  return socket;
}

describe("SignalingClient.waitForOpen", () => {
  test("resolves true immediately when the socket is already open", async () => {
    const client = newClient();
    stubSocket(client, WebSocket.OPEN);

    expect(await client.waitForOpen(1000)).toBe(true);
  });

  test("resolves false when the socket never opens", async () => {
    const client = newClient();
    stubSocket(client, WebSocket.CONNECTING);

    expect(await client.waitForOpen(25)).toBe(false);
  });

  test("resolves false for everyone waiting when the client is closed", async () => {
    const client = newClient();
    stubSocket(client, WebSocket.CONNECTING);

    const waiting = client.waitForOpen(5000);
    client.close();

    expect(await waiting).toBe(false);
  });
});
