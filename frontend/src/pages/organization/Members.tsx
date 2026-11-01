import { Link } from "react-router-dom";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Note } from "@/pages/account/parts";

/* O2, Members and roles (4A.10): the members who take part in a current
   event or took part in a past one, each with their roles. Events come with
   Phase 3, so until then nobody is listed. Invite member comes with
   invitations. */
export default function Members() {
  const { address } = useSession();
  if (address?.kind !== "organization") return null;
  const orgName = address.organization.name;
  return (
    <SignedInShell full>
      <div>
        <Link className="pe-link" to="/">
          Back to {orgName}
        </Link>
      </div>
      <h1 className="pe-page-title">Members and roles</h1>
      <p className="pe-lead">A role gives permissions inside this organization only. Payments are kept separately from roles.</p>
      <div className="pe-card pe-stack pe-empty">
        <h2 className="pe-card__title">No members listed yet</h2>
        <p className="pe-dialog__body">Members who take part in an event of {orgName} appear here, with their roles.</p>
      </div>
      <Note>What each role may do is set in section 7B.6 of the specification.</Note>
    </SignedInShell>
  );
}
