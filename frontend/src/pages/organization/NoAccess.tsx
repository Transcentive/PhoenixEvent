import { Link } from "react-router-dom";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";

/* E3. The same message shows for a missing page, a page in another
   organization and a page that needs another role, so it reveals nothing
   about other organizations. */
export default function NoAccess() {
  const { address } = useSession();
  if (address?.kind !== "organization") return null;
  return (
    <SignedInShell>
      <section className="pe-card pe-stack pe-no-access">
        <h1 className="pe-card__title pe-no-access__title">This page is outside your access</h1>
        <div>
          <span className="pe-badge pe-badge--neutral">
            <svg className="pe-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <rect x="5" y="11" width="14" height="10" rx="2" />
              <path d="M8 11V7a4 4 0 0 1 8 0v4" />
            </svg>
            Private
          </span>
        </div>
        <p className="pe-dialog__body">
          You are logged in to {address.organization.name}. The page you asked for needs a role you do not hold here,
          or it belongs to another organization.
        </p>
        <p className="pe-dialog__body pe-dialog__note">
          To work in another organization, log out and log in to that organization.
        </p>
        <div className="pe-actions pe-actions--apart">
          <Link className="pe-link" to="/logout">
            Log out
          </Link>
          <Link className="pe-btn pe-btn--primary" to="/">
            Go to organization home
          </Link>
        </div>
      </section>
    </SignedInShell>
  );
}
