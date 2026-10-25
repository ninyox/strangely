import { PageList, PageSection, PageShell } from "../components/page-shell";

const SECTIONS = [
  { id: "rules", label: "The rules" },
  { id: "if-someone-breaks-them", label: "If someone breaks them" },
  { id: "what-we-do", label: "What we do about it" },
  { id: "protecting-yourself", label: "Protecting yourself" },
  { id: "adults-only", label: "Adults only" },
];

export default function GuidelinesRoute() {
  return (
    <PageShell
      kicker="Community guidelines"
      title="The rules that make a stranger worth meeting"
      intro="Strangerly connects you to one person at a time. These are the rules that keep that worth doing, and what happens when they are broken."
      sections={SECTIONS}
    >
      <PageSection id="rules" title="The rules">
        <PageList
          items={[
            "Keep it clothed. No nudity and no sexual content, ever.",
            "Keep it kind. No harassment, hate speech, threats, or slurs.",
            "Adults only. If you are under 18, this is not the place for you.",
            "No spam, scams, advertising, or links to get-rich schemes.",
            "Do not record or screenshot the other person. You are talking to a person, not content.",
            "Do not do anything illegal on camera, and do not ask anyone else to.",
          ]}
        />
      </PageSection>

      <PageSection id="if-someone-breaks-them" title="If someone breaks them">
        <p>
          Press <span className="font-medium text-ink">Report</span> during the call. It is available
          the entire time you are connected and for a short while after the call ends, and it is never
          blocked by the five second Skip rule.
        </p>
        <p>
          Press <span className="font-medium text-ink">Stop</span>, or close the tab, if you want out
          immediately. Both work instantly and always.
        </p>
      </PageSection>

      <PageSection id="what-we-do" title="What Strangerly does about reports">
        <p>
          Reports are stored and reviewed by a person. There is no automated banning, no shadow
          moderation, and no real-time content filtering. Nothing about the call is recorded, so a
          report is a description of what happened plus whatever you choose to write — which is why a
          sentence or two helps.
        </p>
        <p>
          Because there are no accounts, somebody who is removed can start again. That is a real
          limitation of anonymous random chat, and pretending otherwise would be dishonest.
        </p>
      </PageSection>

      <PageSection id="protecting-yourself" title="Protecting yourself">
        <PageList
          items={[
            "Assume anything you show can be seen by anyone. Keep identifying details out of frame: documents, post, screens, house numbers.",
            "You do not owe a stranger a conversation. Leaving is always allowed and needs no explanation.",
            "Your IP address is visible to the other person through the peer-to-peer connection.",
            "If somebody is in immediate danger, contact your local emergency services. Strangerly is not a crisis service and cannot dispatch help.",
          ]}
        />
      </PageSection>

      <PageSection id="adults-only" title="Adults only">
        <p>
          Strangerly is intended for people aged 18 and over. There is no identity or age check — the
          site says who it is for and trusts you to respect it. If you are under 18, please do not use
          it.
        </p>
      </PageSection>
    </PageShell>
  );
}
