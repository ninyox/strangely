import type { IceServer, PeerDiagnostics } from "./types";

/**
 * One RTCPeerConnection for one match.
 *
 * The media itself never touches the server: this is a direct peer to peer
 * connection, and the Go backend only ever carries the SDP and ICE messages
 * that set it up. The local stream is owned by the media provider, so closing
 * a call never switches the visitor's own camera off.
 */
export type PeerCallbacks = {
  onRemoteStream: (stream: MediaStream) => void;
  onConnectionState: (state: RTCPeerConnectionState) => void;
  onIceCandidate: (candidate: RTCIceCandidateInit) => void;
};

export class PeerSession {
  private readonly connection: RTCPeerConnection;
  private readonly remote = new MediaStream();
  private pendingCandidates: RTCIceCandidateInit[] = [];
  private remoteDescriptionSet = false;
  private closed = false;

  // Counted so a failed call can say what it saw: candidates that never crossed
  // and candidates that crossed but never paired look identical on screen.
  private localCandidates = 0;
  private remoteCandidates = 0;
  private readonly startedAt = Date.now();

  constructor(
    iceServers: IceServer[],
    localStream: MediaStream,
    private readonly callbacks: PeerCallbacks,
  ) {
    this.connection = new RTCPeerConnection({
      iceServers: iceServers as RTCIceServer[],
      iceCandidatePoolSize: 2,
      bundlePolicy: "max-bundle",
    });

    for (const track of localStream.getTracks()) {
      this.connection.addTrack(track, localStream);
    }

    this.connection.ontrack = (event) => {
      // Track events can fire more than once for the same track; adding a
      // duplicate would make the video element stutter.
      if (!this.remote.getTracks().includes(event.track)) {
        this.remote.addTrack(event.track);
      }
      this.callbacks.onRemoteStream(this.remote);
    };

    this.connection.onicecandidate = (event) => {
      if (event.candidate) {
        this.localCandidates += 1;
        this.callbacks.onIceCandidate(event.candidate.toJSON());
      }
    };

    this.connection.onconnectionstatechange = () => {
      this.callbacks.onConnectionState(this.connection.connectionState);
    };
  }

  get connectionState(): RTCPeerConnectionState {
    return this.connection.connectionState;
  }

  /** The offerer's first step. */
  async createOffer(): Promise<RTCSessionDescriptionInit> {
    const offer = await this.connection.createOffer();
    await this.connection.setLocalDescription(offer);
    return offer;
  }

  /** The answerer's step, on receiving an offer. */
  async acceptOffer(description: RTCSessionDescriptionInit): Promise<RTCSessionDescriptionInit> {
    await this.connection.setRemoteDescription(description);
    this.remoteDescriptionSet = true;
    await this.drainCandidates();

    const answer = await this.connection.createAnswer();
    await this.connection.setLocalDescription(answer);
    return answer;
  }

  /** The offerer's second step, on receiving the answer. */
  async acceptAnswer(description: RTCSessionDescriptionInit): Promise<void> {
    await this.connection.setRemoteDescription(description);
    this.remoteDescriptionSet = true;
    await this.drainCandidates();
  }

  /**
   * ICE candidates that arrive before the remote description are queued: the
   * browser cannot accept them yet, and dropping them would cost seconds of
   * connection time.
   */
  async addIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (this.closed) {
      return;
    }

    this.remoteCandidates += 1;

    if (!this.remoteDescriptionSet) {
      this.pendingCandidates.push(candidate);
      return;
    }

    try {
      await this.connection.addIceCandidate(candidate);
    } catch {
      // A candidate that arrives too late is harmless: ICE keeps trying the
      // others.
    }
  }

  /** What this connection saw, for the log line that explains a failed call. */
  diagnostics(): PeerDiagnostics {
    return {
      ice: this.connection.iceConnectionState,
      connection: this.connection.connectionState,
      localCandidates: this.localCandidates,
      remoteCandidates: this.remoteCandidates,
      remoteDescription: this.remoteDescriptionSet,
      elapsedMs: Date.now() - this.startedAt,
    };
  }

  close(): void {
    if (this.closed) {
      return;
    }
    this.closed = true;

    this.connection.onicecandidate = null;
    this.connection.ontrack = null;
    this.connection.onconnectionstatechange = null;

    for (const sender of this.connection.getSenders()) {
      // Detach the local track without stopping it: the visitor's camera
      // stays on for the next match.
      if (sender.track) {
        try {
          this.connection.removeTrack(sender);
        } catch {
          // Some browsers refuse to remove a track from a closed connection.
        }
      }
    }

    this.connection.close();

    for (const track of this.remote.getTracks()) {
      this.remote.removeTrack(track);
    }
    this.pendingCandidates = [];
  }

  private async drainCandidates(): Promise<void> {
    const queued = this.pendingCandidates;
    this.pendingCandidates = [];

    for (const candidate of queued) {
      try {
        await this.connection.addIceCandidate(candidate);
      } catch {
        // See addIceCandidate.
      }
    }
  }
}

/** Does this browser have everything the call flow needs? */
export function webRtcSupport(): { ok: boolean; message?: string } {
  if (typeof window === "undefined") {
    return { ok: false, message: "WebRTC needs a browser." };
  }
  if (typeof window.RTCPeerConnection !== "function") {
    return {
      ok: false,
      message: "This browser cannot make video calls. Try a current version of Chrome, Firefox, Edge, or Safari.",
    };
  }
  return { ok: true };
}
