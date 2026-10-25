import { PageList, PageSection, PageShell } from "../components/page-shell";

const SECTIONS = [
  { id: "stored", label: "What we store" },
  { id: "never", label: "What we never store" },
  { id: "browser", label: "What your browser keeps" },
  { id: "reports", label: "Reports" },
  { id: "webrtc", label: "WebRTC and anonymity" },
  { id: "infrastructure", label: "Logs and infrastructure" },
  { id: "limits", label: "Honest limitations" },
];

export default function PrivacyRoute() {
  return (
    <PageShell
      kicker="Privacy"
      title="What Strangerly keeps, and what it never sees"
      intro="Written to be read rather than skimmed past. If a claim below were untrue, the product would be a different product."
      sections={SECTIONS}
    >
      <PageSection id="stored" title="What we store">
        <p>
          The minimum needed to run a temporary visit. When you open the site, the server creates an
          anonymous session and gives your browser one opaque, HttpOnly cookie. It contains a random
          identifier and nothing else — no name, no email, no device fingerprint.
        </p>
        <p>
          On the server, the session is held in memory as a hash of that identifier together with the
          display name you typed and the timestamps of the visit. Sessions expire after two hours of
          inactivity by default and are dropped on the next cleanup pass.
        </p>
      </PageSection>

      <PageSection id="never" title="What we never store">
        <PageList
          items={[
            "No video and no audio. Calls are peer-to-peer: media goes from your browser straight to the other person's browser and is never relayed through or recorded by the server.",
            "No call recordings, screenshots, or transcripts.",
            "No chat history, because there is no text chat in this version.",
            "No passwords, emails, or accounts, because there are none.",
            "No IP addresses kept beyond what is operationally necessary. Your address is used in memory to rate-limit session creation; it is not written into the session record or the logs.",
          ]}
        />
      </PageSection>

      <PageSection id="browser" title="What your browser keeps">
        <p>
          Your display name is stored in this browser&apos;s local storage so the field is filled in
          next time. Nothing else is kept locally: the session token lives in an HttpOnly cookie that
          page scripts cannot read, and no media is ever written to disk.
        </p>
      </PageSection>

      <PageSection id="reports" title="Reports">
        <p>
          When you report a call, the server stores the reason, an optional note you write, the match
          it refers to, and the time. It does not store your video, your display name, or any detail
          about the other person. A person reads reports; there is no automated enforcement.
        </p>
      </PageSection>

      <PageSection id="webrtc" title="WebRTC and what it does not promise">
        <p>
          Calls use WebRTC, which is encrypted in transit (DTLS-SRTP) between the two browsers. That
          is not the same as being anonymous: your IP address is visible to the person you are talking
          to, exactly as it would be in any direct connection. A VPN changes what that person can see
          about your connection, but it does not make you unrecognisable to somebody who already knows
          you.
        </p>
      </PageSection>

      <PageSection id="infrastructure" title="Logs and infrastructure">
        <p>
          The service logs operational events — a session was created, a match ended, a report was
          filed — without session tokens, without media, and without the text of any note. If this
          deployment sits behind a reverse proxy or a hosting provider, that layer may keep its own
          access logs including IP addresses. That is a property of the hosting, and it is disclosed
          here rather than hidden.
        </p>
      </PageSection>

      <PageSection id="limits" title="Honest limitations">
        <PageList
          items={[
            "State lives in the memory of a single process. A restart clears every session, queue entry, and match. There is no database and no backup to restore.",
            "Reports are held in memory too, and are dropped when the process restarts or when they age out.",
            "Nobody's identity or age is verified, and the product does not claim otherwise.",
            "There is no cross-device block list, because there are no accounts to attach one to.",
          ]}
        />
      </PageSection>
    </PageShell>
  );
}
