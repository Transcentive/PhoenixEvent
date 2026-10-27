import { Link, Navigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ApiError, getPerson } from "@/api/client";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Alert, Note } from "@/pages/account/parts";

/*
  A9, Profile, shared view (spec 8B.4): what the person shares with the
  viewer. The Invite member button comes with invitations (v1.04).
*/
export default function Person() {
  const { address, session } = useSession();
  const id = useParams().id ?? "";
  const person = useQuery({
    queryKey: ["person", id],
    queryFn: () => getPerson(session!.access_token, id),
    enabled: !!session && id !== session?.account.id,
    retry: false,
  });
  if (address?.kind !== "organization" || !session) return null;
  if (id === session.account.id) return <Navigate to="/profile" replace />;
  const orgName = address.organization.name;
  const p = person.data;
  const items: [string, string | undefined][] = p
    ? [
        ["Personal name", p.personal_name],
        ["Username", p.username],
        ["Email address", p.email],
        ["Roles", p.roles?.join(", ")],
      ]
    : [];

  return (
    <SignedInShell>
      <div>
        <Link className="pe-link" to="/">
          Back to {orgName}
        </Link>
      </div>
      {person.isError ? (
        <Alert>
          {person.error instanceof ApiError && person.error.status === 404
            ? "There is no such person in this organization."
            : "This profile did not load. Reload the page to try again."}
        </Alert>
      ) : p ? (
        <>
          <h1 className="pe-page-title">{p.name}</h1>
          <section className="pe-card pe-stack">
            {items
              .filter(([, value]) => value)
              .map(([label, value]) => (
                <div key={label} className="pe-readonly">
                  <span className="pe-field__label">{label}</span>
                  <span className="pe-readonly__value">{value}</span>
                </div>
              ))}
          </section>
          <Note>
            Read only. Shows what {p.name} shares with you: with station members of a station you both belong to, with
            participants of an event you both take part in, or with members of {orgName}.
          </Note>
        </>
      ) : null}
    </SignedInShell>
  );
}
