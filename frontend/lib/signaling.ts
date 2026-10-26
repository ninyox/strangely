import type { ServerEvent } from "./types";

/**
 * The signaling socket.
 *
 * The Go server relays WebRTC offers, answers and ICE candidates between the
 * two participants of a match, and pushes state changes (matched,
 * partner_left). This class owns the socket, the heartbeat-free reconnect, and
 * nothing else: all decisions live in the call provider.
 */
export type SignalingHandlers = {
  onEvent: (event: ServerEvent) => void;
  onOpen: () => void;
  onClose: (willRetry: boolean) => void;
};

const MAX_RETRIES = 5;
const BASE_DELAY_MS = 500;
const MAX_DELAY_MS = 8000;

export class SignalingClient {
  private socket: WebSocket | null = null;
  private retries = 0;
  private closedByUs = false;
  private timer: ReturnType<typeof setTimeout> | null = null;

  // Callers that are waiting for the socket to open (see waitForOpen).
  private openWaiters: Array<{ resolve: (opened: boolean) => void; timer: ReturnType<typeof setTimeout> }> =
    [];

  constructor(
    private readonly url: string,
    private readonly handlers: SignalingHandlers,
  ) {}

  connect(): void {
    this.closedByUs = false;
    this.clearTimer();

    let socket: WebSocket;
    try {
      socket = new WebSocket(this.url);
    } catch {
      // A malformed URL or a blocked scheme: retrying will not help.
      this.handlers.onClose(false);
      return;
    }

    this.socket = socket;

    socket.onopen = () => {
      this.retries = 0;
      this.settleWaiters(true);
      this.handlers.onOpen();
    };

    socket.onmessage = (message) => {
      if (typeof message.data !== "string") {
        return;
      }
      try {
        this.handlers.onEvent(JSON.parse(message.data) as ServerEvent);
      } catch {
        // A message we cannot parse is ignored: the server never sends one.
      }
    };

    socket.onclose = () => {
      this.socket = null;
      if (this.closedByUs) {
        return;
      }

      const willRetry = this.retries < MAX_RETRIES;
      this.handlers.onClose(willRetry);

      if (willRetry) {
        const delay = Math.min(MAX_DELAY_MS, BASE_DELAY_MS * 2 ** this.retries);
        this.retries += 1;
        this.timer = setTimeout(() => this.connect(), delay);
      } else {
        // Out of retries: nothing is going to open, so nobody waits forever.
        this.settleWaiters(false);
      }
    };

    socket.onerror = () => {
      // The close handler does the work; an error is always followed by one.
    };
  }

  isOpen(): boolean {
    return this.socket?.readyState === WebSocket.OPEN;
  }

  /**
   * Resolve once the socket is open, or with false when it cannot be opened
   * inside the timeout.
   *
   * Matching waits for this. A session that is in the queue (or in a match)
   * without a live socket cannot be reached at all: the server would pair it
   * with somebody, and that somebody would sit in front of a call that can
   * never be answered.
   */
  waitForOpen(timeoutMs = 6000): Promise<boolean> {
    if (this.isOpen()) {
      return Promise.resolve(true);
    }

    return new Promise<boolean>((resolve) => {
      const waiter = {
        resolve,
        timer: setTimeout(() => {
          this.openWaiters = this.openWaiters.filter((entry) => entry !== waiter);
          resolve(false);
        }, timeoutMs),
      };
      this.openWaiters.push(waiter);
    });
  }

  /** Send one message. Anything sent while the socket is down is dropped. */
  send(type: string, matchId: string, payload?: unknown): boolean {
    if (!this.isOpen() || !this.socket) {
      return false;
    }

    this.socket.send(JSON.stringify({ type, matchId, payload }));
    return true;
  }

  close(): void {
    this.closedByUs = true;
    this.clearTimer();
    this.settleWaiters(false);
    this.socket?.close();
    this.socket = null;
  }

  private settleWaiters(opened: boolean) {
    const waiters = this.openWaiters;
    this.openWaiters = [];

    for (const waiter of waiters) {
      clearTimeout(waiter.timer);
      waiter.resolve(opened);
    }
  }

  private clearTimer() {
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }
}
