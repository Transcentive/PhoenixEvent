import { useId, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, createEvent, deleteEvent, getEvent, updateEvent, type BoardOrder, type EventSummary } from "@/api/client";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Alert, Field } from "@/pages/account/parts";
import NoAccess from "@/pages/organization/NoAccess";
import { useIsAdministrator, useMyRoles } from "@/pages/organization/roles";

/* S1, Create event (16B.6). An Organization Administrator creates events
   directly in the first release (7B.6). The name and a board of at least 1
   seat are required; the date, location and description are optional. The
   event is private, and S1 asks which comes first, the board or the chair
   (5E.1). Until S7 and S8 are built, Create event leads back to O1. */
export default function CreateEventPage() {
  const { address } = useSession();
  const roles = useMyRoles();
  const administrator = useIsAdministrator();
  if (address?.kind !== "organization" || !roles.data) return null;
  if (!administrator) return <NoAccess />;
  return <EventForm orgName={address.organization.name} />;
}

/* Edit event (owner's ruling of November 9, 2026): the Create event form,
   filled in. An Organization Administrator, the event's Event Administrator
   and its Event Chair may edit by default; the server decides. The order,
   board first or chair first, shows but does not change: whether it may
   change after creation is open (5E.4). */
export function EditEventPage() {
  const { address, session } = useSession();
  const { id = "" } = useParams();
  const event = useQuery({
    queryKey: ["event", id],
    queryFn: () => getEvent(session!.access_token, id),
    enabled: !!session,
    retry: false,
  });
  if (address?.kind !== "organization") return null;
  if (event.error instanceof ApiError && event.error.status === 404) return <NoAccess />;
  if (!event.data) return null;
  return <EventForm orgName={address.organization.name} existing={event.data} />;
}

const ORDERS: [BoardOrder, string][] = [
  [
    "board_first",
    "The board first. Organization Leadership appoints the board members, and they choose a chair from among themselves.",
  ],
  ["chair_first", "The chair first. Organization Leadership appoints the chair, and the chair appoints the board members."],
];

function EventForm({ orgName, existing }: { orgName: string; existing?: EventSummary }) {
  const { session } = useSession();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [name, setName] = useState(existing?.name ?? "");
  const [date, setDate] = useState(existing?.date ?? "");
  const [location, setLocation] = useState(existing?.location ?? "");
  const [description, setDescription] = useState(existing?.description ?? "");
  const [capacity, setCapacity] = useState(existing ? String(existing.board_capacity) : "");
  const [order, setOrder] = useState<BoardOrder | "">(existing?.board_order ?? "");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const descriptionId = useId();
  const capacityId = useId();

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!session) return;
    setBusy(true);
    try {
      const fields = { name, date, location, description, board_capacity: capacity, board_order: order };
      if (existing) await updateEvent(session.access_token, existing.id, fields);
      else await createEvent(session.access_token, fields);
      await queryClient.invalidateQueries({ queryKey: ["events"] });
      await queryClient.invalidateQueries({ queryKey: ["event"] });
      navigate("/");
    } catch (err) {
      if (err instanceof ApiError && err.fields.length > 0) {
        setFields(Object.fromEntries(err.fields.map((f) => [f.field, f.message])));
        setError(undefined);
      } else {
        setFields({});
        setError(err instanceof ApiError ? err.message : "Something went wrong on our side. Try again.");
      }
      setBusy(false);
    }
  }

  async function remove() {
    if (!session || !existing) return;
    setBusy(true);
    try {
      await deleteEvent(session.access_token, existing.id);
      await queryClient.invalidateQueries({ queryKey: ["events"] });
      navigate("/");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong on our side. Try again.");
      setBusy(false);
    }
  }

  return (
    <SignedInShell>
      <div>
        <Link className="pe-link" to="/">
          Back to {orgName}
        </Link>
      </div>
      <h1 className="pe-page-title">{existing ? "Edit event" : "Create event"}</h1>
      <form className="pe-stack" onSubmit={submit} noValidate>
        {error && <Alert>{error}</Alert>}
        <section className="pe-card pe-stack">
          <Field label="Event name" hint="For example, Halloween Bash." value={name} onChange={setName} error={fields.name} />
          <div className="pe-names">
            <Field label="Date" type="date" value={date} onChange={setDate} error={fields.date} />
            <Field label="Location" value={location} onChange={setLocation} />
          </div>
          <div className="pe-field">
            <label className="pe-field__label" htmlFor={descriptionId}>
              Description
            </label>
            <textarea
              className="pe-input pe-textarea"
              id={descriptionId}
              rows={5}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className={fields.board_capacity ? "pe-field pe-field--error pe-field--short" : "pe-field pe-field--short"}>
            <label className="pe-field__label" htmlFor={capacityId}>
              Event board capacity
            </label>
            <input
              className="pe-input"
              id={capacityId}
              inputMode="numeric"
              value={capacity}
              aria-invalid={fields.board_capacity ? true : undefined}
              onChange={(e) => setCapacity(e.target.value)}
            />
            {fields.board_capacity && <span className="pe-field__error">{fields.board_capacity}</span>}
            <span className="pe-field__hint">Capacity includes the chair.</span>
          </div>
        </section>

        <section className="pe-card pe-stack" aria-labelledby="order-heading">
          <h2 className="pe-card__title" id="order-heading">
            Which comes first, the board or the chair?
          </h2>
          {existing ? (
            <p className="pe-dialog__body">{ORDERS.find(([value]) => value === existing.board_order)?.[1]}</p>
          ) : (
            <fieldset className="pe-choices" aria-labelledby="order-heading">
              {ORDERS.map(([value, label]) => (
                <label key={value} className="pe-choice">
                  <input type="radio" name="board_order" checked={order === value} onChange={() => setOrder(value)} />
                  {label}
                </label>
              ))}
            </fieldset>
          )}
          {fields.board_order && <span className="pe-field__error">{fields.board_order}</span>}
        </section>

        <section className="pe-card pe-stack" aria-labelledby="visibility-heading">
          <h2 className="pe-card__title" id="visibility-heading">
            Who can see this event
          </h2>
          <p className="pe-dialog__body">Private. Members of {orgName} see it after logging in.</p>
        </section>

        {!existing && (
          <p className="pe-text">
            This creates the event for {orgName}. The event starts with no board and no chair.
          </p>
        )}
        <div className="pe-actions">
          <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
            {existing ? "Save event" : "Create event"}
          </button>
          <Link className="pe-btn pe-btn--secondary" to="/">
            Cancel
          </Link>
          {existing && !confirming && (
            <button className="pe-btn pe-btn--danger" type="button" onClick={() => setConfirming(true)}>
              Delete event
            </button>
          )}
        </div>
        {existing && confirming && (
          <section className="pe-card pe-stack" role="alertdialog" aria-labelledby="delete-heading">
            <h2 className="pe-card__title" id="delete-heading">
              Delete {existing.name}?
            </h2>
            <p className="pe-dialog__body">This deletes the event and its board. It cannot be undone.</p>
            <div className="pe-actions">
              <button className="pe-btn pe-btn--secondary" type="button" onClick={() => setConfirming(false)}>
                Keep the event
              </button>
              <button className="pe-btn pe-btn--danger" type="button" disabled={busy} onClick={remove}>
                Delete event
              </button>
            </div>
          </section>
        )}
      </form>
    </SignedInShell>
  );
}
