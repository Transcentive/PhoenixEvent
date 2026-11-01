import { Link } from "react-router-dom";
import { useSession } from "@/session";
import SignedInShell, { BUILDING } from "@/pages/SignedInShell";

/* O1, Organization home. Each link shows only once its page is built;
   events, and Create event or Propose event, come with Phase 3. */
export default function OrgHome() {
  const { address } = useSession();
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
            Members and roles
          </Link>
          <Link className="pe-link" to="/officers">
            Organization officers
          </Link>
        </nav>
      </div>
      <section className="pe-stack">
        <h2 className="pe-section-title">Events</h2>
        <div className="pe-card pe-stack pe-empty">
          <h3 className="pe-card__title">No events yet</h3>
          <p className="pe-dialog__body">Events for {orgName} appear here.</p>
        </div>
      </section>
    </SignedInShell>
  );
}
