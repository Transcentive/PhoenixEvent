import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { getEvents, type EventSummary } from "@/api/client";
import { useSession } from "@/session";
import SignedInShell, { BUILDING } from "@/pages/SignedInShell";
import { Alert } from "@/pages/account/parts";
import { useIsAdministrator } from "./roles";

/* O1, Organization home. Each link shows only once its page is built, and
   only to the roles that may use it (7B.6): Policies and Create event to an
   Organization Administrator. Every member sees the events (16B.6). */
export default function OrgHome() {
  const { address, session } = useSession();
  const administrator = useIsAdministrator();
  const events = useQuery({
    queryKey: ["events"],
    queryFn: () => getEvents(session!.access_token),
    enabled: !!session,
  });
  if (address?.kind !== "organization") return null;
  const orgName = address.organization.name;
  return (
    <SignedInShell full>
      <div className="pe-stack pe-stack--tight">
        <span className="pe-eyebrow">
          {BUILDING}
          Organization
        </span>
        <h1 className="pe-page-title">{orgName}</h1>
        <nav className="pe-links" aria-label={orgName}>
          <Link className="pe-link" to="/members">
            Members
          </Link>
          <Link className="pe-link" to="/officers">
            Organization officers
          </Link>
          {administrator && (
            <Link className="pe-link" to="/policies">
              Policies
            </Link>
          )}
        </nav>
      </div>
      <section className="pe-stack">
        <div className="pe-title-line">
          <h2 className="pe-section-title">Events</h2>
          {administrator && (
            <Link className="pe-btn pe-btn--primary" to="/events/new">
              Create event
            </Link>
          )}
        </div>
        {events.isError && <Alert>The events did not load. Reload the page to try again.</Alert>}
        {events.data?.length === 0 && (
          <div className="pe-card pe-stack pe-empty">
            <h3 className="pe-card__title">No events yet</h3>
            <p className="pe-dialog__body">Events for {orgName} appear here.</p>
          </div>
        )}
        {!!events.data?.length && (
          <div className="pe-event-grid">
            {events.data.map((e) => (
              <EventCard key={e.id} event={e} />
            ))}
          </div>
        )}
      </section>
    </SignedInShell>
  );
}

const LOCK = (
  <svg className="pe-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <rect x="5" y="11" width="14" height="10" rx="2" />
    <path d="M8 11V7a4 4 0 0 1 8 0v4" />
  </svg>
);

/* 2026-10-31 shows as October 31, 2026, with no shift for the time zone. */
export function formatDate(iso: string) {
  const [y = 0, m = 1, d = 1] = iso.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString("en-US", {
    timeZone: "UTC",
    month: "long",
    day: "numeric",
    year: "numeric",
  });
}

/* Every event of the first release is private. Manage event opens the
   Event board (S7, S8), which every member may read. Edit event shows to
   those who may edit (owner's ruling of November 9, 2026). View event shows
   once its page is built. */
function EventCard({ event }: { event: EventSummary }) {
  const when = [event.date && formatDate(event.date), event.location].filter(Boolean).join(" · ");
  return (
    <article className="pe-card pe-stack pe-stack--tight">
      <div className="pe-event-card__head">
        <h3 className="pe-card__title">{event.name}</h3>
        <span className="pe-badge pe-badge--neutral">
          {LOCK}
          Private
        </span>
      </div>
      {when && <p className="pe-dialog__body">{when}</p>}
      <p className="pe-dialog__body">{event.formed ? "Event board formed" : "Event board forming"} · 0 stations</p>
      <div className="pe-actions">
        <Link className="pe-btn pe-btn--secondary" to={`/events/${event.id}/board`}>
          Manage event
        </Link>
        {event.can_edit && (
          <Link className="pe-btn pe-btn--secondary" to={`/events/${event.id}/edit`}>
            Edit event
          </Link>
        )}
      </div>
    </article>
  );
}
