import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { getMembers } from "@/api/client";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Alert, Note } from "@/pages/account/parts";
import { ChangeRoles, useIsAdministrator } from "./roles";

/* O2, Members and roles (4A.10): the members who take part in a current
   event or took part in a past one, each with their roles. An appointment to
   an event board makes a person a participant (5D.1); registrations and
   stations add to the list once they are built. Every member sees it; only an
   Organization Administrator sees Change roles (v1.07). Invite member comes
   with invitations. */
export default function Members() {
  const { address, session } = useSession();
  const administrator = useIsAdministrator();
  const [changing, setChanging] = useState<string | null>(null);
  const members = useQuery({
    queryKey: ["members"],
    queryFn: () => getMembers(session!.access_token),
    enabled: !!session,
  });
  if (address?.kind !== "organization" || !session) return null;
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
      {members.isError && <Alert>The members did not load. Reload the page to try again.</Alert>}
      {members.data?.length === 0 && (
        <div className="pe-card pe-stack pe-empty">
          <h2 className="pe-card__title">No members listed yet</h2>
          <p className="pe-dialog__body">Members who take part in an event of {orgName} appear here, with their roles.</p>
        </div>
      )}
      {!!members.data?.length && (
        <section className="pe-card pe-list" aria-label="Members">
          {members.data.map((m) => (
            <div key={m.id} className="pe-list__item">
              <span className="pe-list__who">
                <Link className="pe-link pe-list__name" to={m.id === session.account.id ? "/profile" : `/people/${m.id}`}>
                  {m.name}
                </Link>
                {m.email && <span className="pe-list__email">{m.email}</span>}
              </span>
              <span className="pe-list__roles">{m.roles.join("; ")}</span>
              {administrator && (
                <button className="pe-btn pe-btn--secondary" type="button" onClick={() => setChanging(m.id)}>
                  Change roles
                </button>
              )}
            </div>
          ))}
        </section>
      )}
      <Note>What each role may do is set in section 7B.6 of the specification.</Note>
      {changing && <ChangeRoles personId={changing} onClose={() => setChanging(null)} />}
    </SignedInShell>
  );
}
