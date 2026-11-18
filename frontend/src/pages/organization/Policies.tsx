import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, getPolicies, setPolicies, type Participation, type Policies } from "@/api/client";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Alert, Notice } from "@/pages/account/parts";
import NoAccess from "./NoAccess";

/* O3, Policies (16B.6). All four sections show. Registration and Event
   Administrator work in the first release; Waiting lists and Lowering a
   station's capacity show, marked as coming in a later release. Only an
   Organization Administrator opens the page (7B.6). */
export default function PoliciesPage() {
  const { address, session } = useSession();
  const policies = useQuery({
    queryKey: ["policies"],
    queryFn: () => getPolicies(session!.access_token),
    enabled: !!session,
    retry: false,
  });
  if (address?.kind !== "organization" || !session) return null;
  if (policies.error instanceof ApiError && policies.error.status === 404) return <NoAccess />;
  const orgName = address.organization.name;
  return (
    <SignedInShell>
      <div>
        <Link className="pe-link" to="/">
          Back to {orgName}
        </Link>
      </div>
      <h1 className="pe-page-title">Policies</h1>
      {policies.isError && <Alert>The policies did not load. Reload the page to try again.</Alert>}
      {policies.data && <PoliciesForm initial={policies.data} />}
    </SignedInShell>
  );
}

const PARTICIPATION: [Participation, string][] = [
  ["subscribers", "Subscribers. A person pays to participate."],
  ["registrants", "Registrants. A person registers to participate."],
  ["both", "Both. A person either registers or pays."],
];

const LATER = "This setting comes in a later release.";

function PoliciesForm({ initial }: { initial: Policies }) {
  const { session } = useSession();
  const queryClient = useQueryClient();
  const [approve, setApprove] = useState(initial.approve_registrations);
  const [participation, setParticipation] = useState<Participation>(initial.participation);
  const [holdsSeat, setHoldsSeat] = useState(initial.event_admin_holds_seat);
  const [error, setError] = useState<string>();
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);

  async function save(e: FormEvent) {
    e.preventDefault();
    if (!session) return;
    setBusy(true);
    setSaved(false);
    try {
      const p = await setPolicies(session.access_token, {
        approve_registrations: approve,
        participation,
        event_admin_holds_seat: holdsSeat,
      });
      queryClient.setQueryData(["policies"], p);
      setError(undefined);
      setSaved(true);
    } catch (err) {
      setError(
        err instanceof ApiError ? (err.fields[0]?.message ?? err.message) : "Something went wrong on our side. Try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="pe-stack" onSubmit={save} noValidate>
      {error && <Alert>{error}</Alert>}
      {saved && <Notice>Policies saved.</Notice>}

      <section className="pe-card pe-stack" aria-labelledby="registration-heading">
        <h2 className="pe-card__title" id="registration-heading">
          Registration
        </h2>
        <label className="pe-choice">
          <input type="checkbox" checked={approve} onChange={(e) => setApprove(e.target.checked)} />
          Approve each registration before the person chooses stations
        </label>
        <fieldset className="pe-choices">
          <legend className="pe-field__label">Who participates in an event</legend>
          {PARTICIPATION.map(([value, label]) => (
            <label key={value} className="pe-choice">
              <input
                type="radio"
                name="participation"
                checked={participation === value}
                onChange={() => setParticipation(value)}
              />
              {label}
            </label>
          ))}
        </fieldset>
        <p className="pe-note">A person becomes a participant of an event first. A participant then signs up for a station.</p>
      </section>

      <section className="pe-card pe-stack" aria-labelledby="waiting-heading">
        <h2 className="pe-card__title" id="waiting-heading">
          Waiting lists
        </h2>
        <p className="pe-dialog__body">Set this before an event board forms. The board's own choice depends on yours.</p>
        <fieldset className="pe-choices" disabled>
          <legend className="pe-field__label">Organization determination</legend>
          <label className="pe-choice">
            <input type="radio" name="waiting" />
            Prohibit waiting lists. The event board keeps them off.
          </label>
          <label className="pe-choice">
            <input type="radio" name="waiting" />
            Request waiting lists. The event board may turn them off.
          </label>
          <label className="pe-choice">
            <input type="radio" name="waiting" defaultChecked />
            Leave it to the event board. The board may turn them on or off.
          </label>
        </fieldset>
        <p className="pe-note">{LATER}</p>
      </section>

      <section className="pe-card pe-stack" aria-labelledby="admin-heading">
        <h2 className="pe-card__title" id="admin-heading">
          Event Administrator
        </h2>
        <p className="pe-dialog__body">Whether an event's Event Administrator holds a seat on its event board.</p>
        <fieldset className="pe-choices">
          <legend className="pe-field__label">For every event</legend>
          <label className="pe-choice">
            <input type="radio" name="seat" checked={holdsSeat} onChange={() => setHoldsSeat(true)} />
            Holds a seat. The Event Administrator counts toward the board's seats.
          </label>
          <label className="pe-choice">
            <input type="radio" name="seat" checked={!holdsSeat} onChange={() => setHoldsSeat(false)} />
            Holds no seat. The Event Administrator works alongside the board. (Default)
          </label>
        </fieldset>
        <p className="pe-note">Organization Leadership appoints each event's Event Administrator.</p>
      </section>

      <section className="pe-card pe-stack" aria-labelledby="capacity-heading">
        <h2 className="pe-card__title" id="capacity-heading">
          Lowering a station's capacity
        </h2>
        <p className="pe-dialog__body">What happens when an event board sets fewer spots than the people already in a station.</p>
        <fieldset className="pe-choices" disabled>
          <legend className="pe-field__label">For every event</legend>
          <label className="pe-choice">
            <input type="radio" name="capacity" defaultChecked />
            Refuse. The board removes people first, then lowers the number. (Default)
          </label>
          <label className="pe-choice">
            <input type="radio" name="capacity" />
            Allow, and remove no one. The station shows Full until enough people leave.
          </label>
          <label className="pe-choice">
            <input type="radio" name="capacity" />
            Allow, and remove the people who joined most recently. Each is emailed.
          </label>
          <label className="pe-choice">
            <input type="checkbox" />
            Fix this for every event. Event boards cannot change it.
          </label>
        </fieldset>
        <p className="pe-note">{LATER} The first release always refuses.</p>
      </section>

      <div className="pe-actions">
        <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
          Save policies
        </button>
        <Link className="pe-btn pe-btn--secondary" to="/">
          Cancel
        </Link>
      </div>
    </form>
  );
}
