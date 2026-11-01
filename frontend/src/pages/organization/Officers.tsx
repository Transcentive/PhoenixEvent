import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { getOfficers } from "@/api/client";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Alert, Note } from "@/pages/account/parts";
import { ChangeRoles, useIsAdministrator } from "./roles";

/* O8, Organization officers (4A.10): everyone who holds Organization
   Leadership or Organization Administrator. Every member sees it; only an
   Organization Administrator sees Change roles (v1.07). */
export default function Officers() {
  const { address, session } = useSession();
  const administrator = useIsAdministrator();
  const [changing, setChanging] = useState<string | null>(null);
  const officers = useQuery({
    queryKey: ["officers"],
    queryFn: () => getOfficers(session!.access_token),
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
      <h1 className="pe-page-title">Organization officers</h1>
      <p className="pe-lead">The people who hold Organization Leadership or Organization Administrator in {orgName}.</p>
      {officers.isError && <Alert>The officers did not load. Reload the page to try again.</Alert>}
      {officers.data && (
        <section className="pe-card pe-list">
          {officers.data.map((o) => (
            <div key={o.id} className="pe-list__item">
              <Link className="pe-link pe-list__name" to={o.id === session.account.id ? "/profile" : `/people/${o.id}`}>
                {o.name}
              </Link>
              <span className="pe-list__roles">{o.roles.join(", ")}</span>
              {administrator && (
                <button className="pe-btn pe-btn--secondary" type="button" onClick={() => setChanging(o.id)}>
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
