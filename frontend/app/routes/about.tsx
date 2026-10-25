import { PageList, PageSection, PageShell } from "../components/page-shell";

const SECTIONS = [
  { id: "what-this-is", label: "What this is" },
  { id: "how-matching-works", label: "How matching works" },
  { id: "five-second-rule", label: "Why five seconds" },
  { id: "what-it-is-not", label: "What it is not" },
  { id: "built-with", label: "How it is built" },
];

export default function AboutRoute() {
  return (
    <PageShell
      kicker="About"
      title="One click away from someone new"
      intro="A small, deliberately plain random video chat. No feed, no followers, no profile to maintain — just one conversation with one person, and nothing kept afterwards."
      sections={SECTIONS}
    >
      <PageSection id="what-this-is" title="What this is">
        <p>
          You choose a name for the visit, turn on your camera, press start, and you are paired with
          one other person who is waiting. That is the whole product. There is no account to create,
          no history to scroll, and no way for either of you to find the other again.
        </p>
      </PageSection>

      <PageSection id="how-matching-works" title="How matching works">
        <p>
          Sessions wait in a single queue and are paired in the order they arrived. When two people
          are matched, the server tells each browser who the other one is and then steps out of the
          way: the video travels directly between the two browsers, not through this server. The
          server keeps only what it needs — who is waiting, who is talking to whom, and when the call
          started.
        </p>
        <p>
          One side is designated to make the connection offer, so both browsers never negotiate at
          once. The server also relays the few small messages that set the connection up, which is why
          the state you see (connecting, connected, ended) comes from the server rather than from your
          browser guessing.
        </p>
      </PageSection>

      <PageSection id="five-second-rule" title="Why Skip is locked for five seconds">
        <p>
          Skip is locked for the first five seconds of a call so neither person can be discarded
          before they have even been seen. The lock is enforced by the server, not by the button, and
          it applies to Skip alone: stopping the call, muting, turning the camera off, and reporting
          all work from the first second.
        </p>
        <p>
          The countdown you see is calculated from the server&apos;s own match timestamp, so a wrong
          clock in your browser — or one that is deliberately moved — changes nothing about when the
          rule lifts.
        </p>
      </PageSection>

      <PageSection id="what-it-is-not" title="What it is not">
        <PageList
          items={[
            "Not moderated in real time. Reports are read by a person, later.",
            "Not anonymous in the absolute sense: your IP address is visible to the person you are talking to.",
            "Not persistent. No friends, no history, no way to find someone again.",
            "Not a dating service, a marketplace, or a place to advertise.",
            "Not for anyone under 18.",
          ]}
        />
      </PageSection>

      <PageSection id="built-with" title="How it is built">
        <p>
          A React single-page frontend (React Router, TypeScript, Tailwind) and a Go backend (Gin,
          gorilla/websocket) that handles anonymous sessions, matchmaking, signaling, and reporting.
          Media never touches the server. There is no database in this version: matchmaking state
          lives in the memory of one process, which is why a restart clears the queue.
        </p>
      </PageSection>
    </PageShell>
  );
}
