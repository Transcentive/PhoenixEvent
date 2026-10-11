import { useQuery } from "@tanstack/react-query";
import { getHealth } from "@/api/client";

/*
  The one page of Phase 1. It shows that the web client reaches the API and
  the API reaches the database, which is the health check the phase ends on.
  Phase 2 replaces it with the account screens (A1 to A7).
*/

type State = "checking" | "ready" | "unavailable";

const BADGES: Record<State, { word: string; role: string; icon: JSX.Element }> = {
  checking: {
    word: "Checking",
    role: "caution",
    icon: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="M12 7v5l3 2" />
      </>
    ),
  },
  ready: {
    word: "Ready",
    role: "positive",
    icon: (
      <>
        <circle cx="12" cy="12" r="9" />
        <path d="M8 12.5l2.5 2.5L16 9.5" />
      </>
    ),
  },
  unavailable: {
    word: "Unavailable",
    role: "critical",
    icon: (
      <>
        <path d="M12 4l9 16H3z" />
        <path d="M12 10v4M12 17v.5" />
      </>
    ),
  },
};

function StatusBadge({ state }: { state: State }) {
  const badge = BADGES[state];
  return (
    <span className={`pe-badge pe-badge--${badge.role}`}>
      <svg
        className="pe-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        {badge.icon}
      </svg>
      {badge.word}
    </span>
  );
}

export default function FoundationPage() {
  const health = useQuery({ queryKey: ["health"], queryFn: getHealth, retry: false });

  let state: State = "checking";
  let note = "Asking the API.";
  if (health.isError) {
    state = "unavailable";
    note = "The API did not answer. Start it, then reload this page.";
  } else if (health.data) {
    if (health.data.status === "ok") {
      state = "ready";
      note = `The API and the database both answer. Build ${health.data.build}.`;
    } else {
      state = "unavailable";
      note = "The API answers and the database does not. Start PostgreSQL, then reload this page.";
    }
  }

  return (
    <main className="pe-page">
      <header>
        <h1 className="pe-page__name">PhoenixEvent</h1>
        <p className="pe-page__lead">People working an event together, station by station.</p>
      </header>
      <section className="pe-card" aria-labelledby="status-label">
        <div className="pe-status">
          <h2 className="pe-status__label" id="status-label">
            API and database
          </h2>
          <StatusBadge state={state} />
        </div>
        <p className="pe-status__note" role="status">
          {note}
        </p>
      </section>
    </main>
  );
}
