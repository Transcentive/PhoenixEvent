import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ApiError,
  appoint,
  closeVoting,
  decideRemoval,
  editRemoval,
  getBoard,
  recordChair,
  recordDeputy,
  removeFromBoard,
  replaceChair,
  startRemoval,
  voteOnRemoval,
  withdrawRemoval,
  type Board,
  type Position,
  type Removal,
  type ReplaceWhy,
  type Seat,
} from "@/api/client";
import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Alert, Notice } from "@/pages/account/parts";
import NoAccess from "@/pages/organization/NoAccess";

/* Event board: forming the board first (S7) or the chair first (S8), with
   Appoint board member (D1). Every member reads it; each action shows only to
   those 7B.6 and 7B.7 name, and the server checks again. */
export default function EventBoardPage() {
  const { address, session } = useSession();
  const { id = "" } = useParams();
  const board = useQuery({
    queryKey: ["board", id],
    queryFn: () => getBoard(session!.access_token, id),
    enabled: !!session,
    retry: false,
  });
  if (address?.kind !== "organization") return null;
  if (board.error instanceof ApiError && board.error.status === 404) return <NoAccess />;
  const orgName = address.organization.name;
  return (
    <SignedInShell full>
      <div>
        <Link className="pe-link" to="/">
          Back to {orgName}
        </Link>
      </div>
      {board.isError && <Alert>The event board did not load. Reload the page to try again.</Alert>}
      {board.data && <BoardView board={board.data} orgName={orgName} />}
    </SignedInShell>
  );
}

const CLOCK = (
  <svg className="pe-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <circle cx="12" cy="12" r="9" />
    <path d="M12 7v5l3 2" />
  </svg>
);

function Status({ children, done }: { children: string; done?: boolean }) {
  return (
    <span className={done ? "pe-badge pe-badge--positive" : "pe-badge pe-badge--neutral"}>
      {!done && CLOCK}
      {children}
    </span>
  );
}

const POSITION: Record<Position, string> = {
  chair: "Event Chair",
  member: "Board Member",
  event_administrator: "Event Administrator",
};

function useBoardAction(eventId: string) {
  const queryClient = useQueryClient();
  return (b: Board) => {
    queryClient.setQueryData(["board", eventId], b);
    void queryClient.invalidateQueries({ queryKey: ["events"] });
  };
}

function message(err: unknown) {
  if (err instanceof ApiError) return err.fields[0]?.message ?? err.message;
  return "Something went wrong on our side. Try again.";
}

function BoardView({ board, orgName }: { board: Board; orgName: string }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const [dialog, setDialog] = useState(false);
  const [removing, setRemoving] = useState<Seat | null>(null);
  const [replacing, setReplacing] = useState(false);
  const e = board.event;
  const boardFirst = e.board_order === "board_first";
  const chair = board.seats.find((s) => s.position === "chair");
  const members = board.seats.filter((s) => s.position === "member");
  const admin = board.seats.find((s) => s.position === "event_administrator");
  const remove = useMutation({
    mutationFn: (person: string) => removeFromBoard(session!.access_token, e.id, person),
    onSuccess: store,
  });

  const step1Done = boardFirst ? members.length > 0 : !!chair;
  const step2Done = boardFirst ? !!chair : members.length > 0;
  const full = board.seats_taken >= e.board_capacity;
  const ratio = Math.min(100, Math.round((board.seats_taken / e.board_capacity) * 100));

  const membersCard = (
    <section className="pe-card pe-stack" aria-labelledby="members-heading">
      <div className="pe-title-line">
        <h2 className="pe-card__title" id="members-heading">
          Board members
        </h2>
        {(board.can_appoint_members || board.can_appoint_admin) && (
          <button className="pe-btn pe-btn--primary" type="button" onClick={() => setDialog(true)}>
            Appoint board member
          </button>
        )}
      </div>
      <div className="pe-meter">
        <div className="pe-meter__track" aria-hidden="true">
          <div className="pe-meter__fill" style={{ width: `${ratio}%` }} />
        </div>
        <div className="pe-meter__row">
          <span>
            {board.seats_taken} of {e.board_capacity} seats taken
          </span>
          <span className="pe-meter__note">Includes the chair</span>
        </div>
      </div>
      {remove.isError && <Alert>{message(remove.error)}</Alert>}
      {board.seats.length > 0 && (
        <div className="pe-list">
          {[...(chair ? [chair] : []), ...members, ...(admin ? [admin] : [])].map((s) => (
            <div key={s.id} className="pe-list__item">
              <span className="pe-list__name">{s.name}</span>
              <span className="pe-list__roles">
                {POSITION[s.position]}
                {s.position === "event_administrator" && !board.event_admin_holds_seat && " · holds no seat"}
                {s.deputy && " · Deputy Board Chair"}
              </span>
              {board.can_remove && s.position === "event_administrator" && (
                <button className="pe-btn pe-btn--secondary" type="button" onClick={() => remove.mutate(s.id)}>
                  Remove
                </button>
              )}
              {board.can_start_removal.includes(s.id) && !board.removals.some((r) => r.person === s.id) && (
                <button className="pe-btn pe-btn--secondary" type="button" onClick={() => setRemoving(s)}>
                  Start removal
                </button>
              )}
            </div>
          ))}
        </div>
      )}
      {!boardFirst && !chair && (
        <p className="pe-dialog__body">
          The chair appoints the board members. Once a chair is appointed, the chair takes 1 of the {e.board_capacity}{" "}
          seats and sees "Appoint board member" here.
        </p>
      )}
      {/* "Appointing a person to the board makes them a participant of the
          Event" belongs in the specification only, not on this page (owner,
          November 26, 2026; 5D.1). */}
      {boardFirst && !chair && <p className="pe-note">Organization Leadership acts for the Organization here.</p>}
    </section>
  );

  const chairCard = (
    <section className="pe-card pe-stack" aria-labelledby="chair-heading">
      <h2 className="pe-card__title" id="chair-heading">
        Chair
      </h2>
      {chair ? (
        <>
          <p className="pe-dialog__body">
            <strong>{chair.name}</strong> is the Event Chair.
          </p>
          {board.can_replace_chair && (
            <div>
              <button className="pe-btn pe-btn--secondary" type="button" onClick={() => setReplacing(true)}>
                Replace chair
              </button>
            </div>
          )}
        </>
      ) : (
        <>
          <div>
            <Status>No chair yet</Status>
          </div>
          {boardFirst ? (
            <RecordChair board={board} />
          ) : (
            <AppointChair board={board} orgName={orgName} />
          )}
        </>
      )}
    </section>
  );

  return (
    <>
      <div className="pe-stack pe-stack--tight">
        <span className="pe-eyebrow">Event · {e.name}</span>
        <h1 className="pe-page-title">Event board</h1>
        <p className="pe-lead">
          The Organization created {e.name}.{" "}
          {boardFirst
            ? "This event forms its board first, and the board members then choose the chair."
            : "This event appoints its chair first, and the chair then appoints the board members."}
        </p>
        <p className="pe-note">
          {board.formed
            ? `The board of ${e.name} is formed.`
            : `Registration for ${e.name} opens once the board is formed. The board is formed once there is a chair and board members.`}
        </p>
        {/* Organization Leadership forms the board; can_appoint_admin is true
            exactly when the signed-in person holds it. */}
        {!board.can_appoint_admin && !step1Done && (
          <Notice>
            You need Organization Leadership to {boardFirst ? "appoint the board members" : "appoint the chair"}. An
            Organization Administrator can give you that role with Change roles on your member profile.
          </Notice>
        )}
        {e.can_edit && (
          <div>
            <Link className="pe-link" to={`/events/${e.id}/edit`}>
              Edit event
            </Link>
          </div>
        )}
      </div>

      <section className="pe-card pe-stack" aria-labelledby="forming-heading">
        <h2 className="pe-card__title" id="forming-heading">
          Forming the board
        </h2>
        <div className="pe-list">
          <div className="pe-list__item">
            <span className="pe-list__name">
              <strong>Step 1.</strong>{" "}
              {boardFirst ? "The Organization appoints the board members." : "The Organization appoints the chair."}
            </span>
            <Status done={step1Done}>{step1Done ? "Done" : "In progress"}</Status>
          </div>
          <div className="pe-list__item">
            <span className="pe-list__name">
              <strong>Step 2.</strong>{" "}
              {boardFirst ? "The board members choose a chair from among themselves." : "The chair appoints the board members."}
            </span>
            <Status done={step2Done}>{step2Done ? "Done" : step1Done ? "In progress" : "Waiting for step 1"}</Status>
          </div>
        </div>
      </section>

      {/* Board members on the left; the Event Chair on the right with the
          Deputy Board Chair below it, whichever comes first (owner, November
          25 and 26, 2026). On a phone: Chair, Deputy Board Chair, then Board
          members. */}
      <div className="pe-board-grid">
        <div className="pe-board-grid__main">{membersCard}</div>
        <div className="pe-board-grid__side pe-stack">
          {chairCard}
          {members.length > 0 && <Deputy board={board} />}
        </div>
      </div>

      {board.removals.length > 0 && <Removals board={board} />}

      {dialog && <AppointDialog board={board} full={full} onClose={() => setDialog(false)} />}
      {removing && <StartRemovalDialog board={board} seat={removing} onClose={() => setRemoving(null)} />}
      {replacing && chair && <ReplaceChairDialog board={board} chair={chair} onClose={() => setReplacing(false)} />}
    </>
  );
}

function AppointChair({ board, orgName }: { board: Board; orgName: string }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const [person, setPerson] = useState("");
  const id = useId();
  const save = useMutation({
    mutationFn: () => appoint(session!.access_token, board.event.id, person, "chair"),
    onSuccess: store,
  });
  if (!board.can_appoint_chair) {
    return <p className="pe-dialog__body">Organization Leadership appoints the chair.</p>;
  }
  return (
    <div className="pe-stack">
      {save.isError && <Alert>{message(save.error)}</Alert>}
      <div className="pe-field">
        <label className="pe-field__label" htmlFor={id}>
          Member to appoint as chair
        </label>
        <select className="pe-input" id={id} value={person} onChange={(e) => setPerson(e.target.value)}>
          <option value="">Choose a member</option>
          {board.candidates.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        {/* That becoming chair makes a participant is in the specification
            only (owner, November 26, 2026; 5D.1). */}
        <span className="pe-field__hint">The list holds members of {orgName}.</span>
      </div>
      <div>
        <button className="pe-btn pe-btn--primary" type="button" disabled={!person || save.isPending} onClick={() => save.mutate()}>
          Appoint chair
        </button>
      </div>
      <p className="pe-note">Organization Leadership acts for the Organization here.</p>
    </div>
  );
}

function RecordChair({ board }: { board: Board }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const [person, setPerson] = useState("");
  const id = useId();
  const members = board.seats.filter((s) => s.position === "member");
  const save = useMutation({
    mutationFn: () => recordChair(session!.access_token, board.event.id, person),
    onSuccess: store,
  });
  return (
    <div className="pe-stack">
      <p className="pe-dialog__body">The board members choose the chair from among themselves.</p>
      {board.can_record_chair && (
        <>
          {save.isError && <Alert>{message(save.error)}</Alert>}
          <div className="pe-field">
            <label className="pe-field__label" htmlFor={id}>
              The board's choice
            </label>
            <select className="pe-input" id={id} value={person} onChange={(e) => setPerson(e.target.value)}>
              <option value="">Choose a board member</option>
              {members.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <button className="pe-btn pe-btn--secondary" type="button" disabled={!person || save.isPending} onClick={() => save.mutate()}>
              Record the board's choice
            </button>
          </div>
        </>
      )}
      <p className="pe-note">
        The board members agree on their chair among themselves. Organization Leadership records the choice here. The
        choice may be recorded before every seat is filled.
      </p>
    </div>
  );
}

/* Why the signed-in person may not appoint a role, following 5E.4, 7B.6 and
   7B.7; null when they may. */
function whyNot(board: Board, role: Position): string | null {
  const hasChair = board.seats.some((s) => s.position === "chair");
  const boardFirst = board.event.board_order === "board_first";
  if (role === "chair") {
    if (board.can_appoint_chair) return null;
    if (hasChair) return "the seat is filled";
    if (boardFirst) return "the board members choose the chair";
    return "only Organization Leadership appoints the chair";
  }
  if (role === "member") {
    if (board.can_appoint_members) return null;
    if (hasChair) return "only the Event Chair appoints Board Members";
    if (!boardFirst) return "the Event Chair appoints Board Members, once appointed";
    return "only Organization Leadership appoints Board Members";
  }
  return board.can_appoint_admin ? null : "only Organization Leadership appoints the Event Administrator";
}

/* D1, Appoint board member. All three roles are listed; those the signed-in
   person may not appoint are greyed out, with the reason (owner, November 18,
   2026). */
function AppointDialog({ board, full, onClose }: { board: Board; full: boolean; onClose: () => void }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const roles = (["chair", "member", "event_administrator"] as Position[]).map((r) => ({ role: r, why: whyNot(board, r) }));
  const [person, setPerson] = useState("");
  const [position, setPosition] = useState<Position>(roles.find((r) => !r.why)?.role ?? "member");
  const personId = useId();
  const roleId = useId();
  const save = useMutation({
    mutationFn: () => appoint(session!.access_token, board.event.id, person, position),
    onSuccess: (b) => {
      store(b);
      onClose();
    },
  });
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  const e = board.event;
  const lastSeat = board.seats_taken + 1 === e.board_capacity;
  return (
    <div className="pe-scrim pe-overlay" onClick={(ev) => ev.target === ev.currentTarget && onClose()}>
      <div className="pe-dialog" role="dialog" aria-modal="true" aria-labelledby="appoint-title">
        <h1 className="pe-dialog__title" id="appoint-title">
          Appoint board member
        </h1>
        <p className="pe-dialog__body">
          Event board for {e.name}. {board.seats_taken} of {e.board_capacity} seats taken, including the chair.
          {full ? " Every seat is taken." : lastSeat ? " This appointment fills the board." : ""}
        </p>
        {save.isError && <Alert>{message(save.error)}</Alert>}
        <div className="pe-field">
          <label className="pe-field__label" htmlFor={personId}>
            Member
          </label>
          <select className="pe-input" id={personId} value={person} onChange={(ev) => setPerson(ev.target.value)}>
            <option value="">Choose a member</option>
            {board.candidates.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          {/* That appointing makes a participant is in the specification only
              (owner, November 26, 2026; 5D.1). */}
        </div>
        <div className="pe-field">
          <label className="pe-field__label" htmlFor={roleId}>
            Role on the board
          </label>
          <select className="pe-input" id={roleId} value={position} onChange={(ev) => setPosition(ev.target.value as Position)}>
            {roles.map(({ role, why }) => (
              <option key={role} value={role} disabled={!!why}>
                {why ? `${POSITION[role]} (${why})` : POSITION[role]}
              </option>
            ))}
          </select>
          <span className="pe-field__hint">
            Whether an Event Administrator holds a seat on the board follows the organization's Policies. Only
            Organization Leadership may appoint the Event Administrator; appointing one replaces the one before.
          </span>
        </div>
        <div className="pe-dialog__actions">
          <button className="pe-btn pe-btn--secondary" type="button" onClick={onClose}>
            Cancel
          </button>
          <button className="pe-btn pe-btn--primary" type="button" disabled={!person || save.isPending} onClick={() => save.mutate()}>
            Appoint board member
          </button>
        </div>
      </div>
    </div>
  );
}

/* 5D.3: what a removal needs, in the owner's words. */
function needs(board: Board, r: Removal) {
  if (r.position === "chair")
    return "Terminating the Event Chair needs a unanimous vote of the Event Board, the Event Chair recused, and approval from an Organization Leadership agent.";
  if (r.needs_chair)
    return "Terminating a Board Member needs Organization Leadership and the Event Chair to agree.";
  return `Before ${board.event.name} has an Event Chair, removing a Board Member needs a unanimous vote of the Event Board, that member recused, and approval from an Organization Leadership agent.`;
}

/* The open removals, each with its votes, approvals and the actions open to
   the person viewing. */
function Removals({ board }: { board: Board }) {
  return (
    <section className="pe-card pe-stack" aria-labelledby="removals-heading">
      <h2 className="pe-card__title" id="removals-heading">
        Removals
      </h2>
      {board.removals.map((r) => (
        <RemovalItem key={r.id} board={board} removal={r} />
      ))}
    </section>
  );
}

function RemovalItem({ board, removal: r }: { board: Board; removal: Removal }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const token = session!.access_token;
  const e = board.event;
  const [editing, setEditing] = useState(false);
  const [reason, setReason] = useState(r.reason);
  const reasonId = useId();
  const act = useMutation({
    mutationFn: (f: () => Promise<Board>) => f(),
    onSuccess: (b) => {
      store(b);
      setEditing(false);
    },
  });
  const run = (f: () => Promise<Board>) => act.mutate(f);
  return (
    <div className="pe-stack pe-stack--tight">
      <p className="pe-dialog__body">
        <strong>Removal of {r.name}</strong>, {POSITION[r.position]}. Started by {r.started_by}.
        {r.reason && <> Reason: {r.reason}</>}
      </p>
      <p className="pe-note">{needs(board, r)}</p>
      {r.needs_vote && !r.voting_closed && (
        <p className="pe-note">
          Votes may change until the Event Board closes voting. The Event Chair records the close; before {e.name} has an
          Event Chair, Organization Leadership does. Another removal of the same person is a new round. A Board Member who has not voted at the close is
          abstaining; if anyone abstains, the Event Board decides whether the vote carries, and the close records it.
        </p>
      )}
      <div className="pe-list">
        {r.needs_vote && (
          <div className="pe-list__item">
            <span className="pe-list__name">Board vote{r.voting_closed ? " (closed)" : " (open)"}</span>
            <span className="pe-list__roles">
              {r.votes.map((v) => `${v.name}: ${v.in_favor ? "in favor" : "against"}`).join(" · ")}
              {r.waiting.length > 0 && `${r.votes.length ? " · " : ""}Not voted (abstaining at the close): ${r.waiting.join(", ")}`}
            </span>
          </div>
        )}
        {r.needs_chair && (
          <div className="pe-list__item">
            <span className="pe-list__name">Event Chair</span>
            <Status done={r.chair_agreed}>{r.chair_agreed ? "Agreed" : "Waiting"}</Status>
          </div>
        )}
        <div className="pe-list__item">
          <span className="pe-list__name">Organization Leadership</span>
          <Status done={r.leadership_approved}>{r.leadership_approved ? "Approved" : "Waiting"}</Status>
        </div>
      </div>
      {act.isError && <Alert>{message(act.error)}</Alert>}
      {editing && (
        <div className="pe-field">
          <label className="pe-field__label" htmlFor={reasonId}>
            Reason
          </label>
          <textarea className="pe-input" id={reasonId} value={reason} onChange={(ev) => setReason(ev.target.value)} />
        </div>
      )}
      <div className="pe-dialog__actions">
        {r.can_vote && r.my_vote !== false && (
          <button className="pe-btn pe-btn--secondary" type="button" disabled={act.isPending} onClick={() => run(() => voteOnRemoval(token, e.id, r.id, false))}>
            {r.my_vote === true ? "Change vote to against" : "Vote against"}
          </button>
        )}
        {r.can_vote && r.my_vote !== true && (
          <button className="pe-btn pe-btn--primary" type="button" disabled={act.isPending} onClick={() => run(() => voteOnRemoval(token, e.id, r.id, true))}>
            {r.my_vote === false ? "Change vote to in favor" : "Vote in favor"}
          </button>
        )}
        {r.can_close_voting &&
          (r.waiting.length > 0 && r.votes.every((v) => v.in_favor) ? (
            <>
              <button className="pe-btn pe-btn--secondary" type="button" disabled={act.isPending} onClick={() => run(() => closeVoting(token, e.id, r.id, false))}>
                Close voting: does not carry
              </button>
              <button className="pe-btn pe-btn--secondary" type="button" disabled={act.isPending} onClick={() => run(() => closeVoting(token, e.id, r.id, true))}>
                Close voting: carries
              </button>
            </>
          ) : (
            <button className="pe-btn pe-btn--secondary" type="button" disabled={act.isPending} onClick={() => run(() => closeVoting(token, e.id, r.id))}>
              Close voting
            </button>
          ))}
        {(r.can_approve || r.can_agree) && (
          <>
            <button className="pe-btn pe-btn--secondary" type="button" disabled={act.isPending} onClick={() => run(() => decideRemoval(token, e.id, r.id, false))}>
              Decline
            </button>
            <button className="pe-btn pe-btn--primary" type="button" disabled={act.isPending} onClick={() => run(() => decideRemoval(token, e.id, r.id, true))}>
              {r.can_approve ? "Approve removal" : "Agree to removal"}
            </button>
          </>
        )}
        {r.can_edit &&
          (editing ? (
            <>
              <button className="pe-btn pe-btn--secondary" type="button" onClick={() => setEditing(false)}>
                Cancel
              </button>
              <button className="pe-btn pe-btn--primary" type="button" disabled={act.isPending} onClick={() => run(() => editRemoval(token, e.id, r.id, reason))}>
                Save reason
              </button>
            </>
          ) : (
            <>
              <button className="pe-btn pe-btn--secondary" type="button" onClick={() => setEditing(true)}>
                Edit reason
              </button>
              <button className="pe-btn pe-btn--danger" type="button" disabled={act.isPending} onClick={() => run(() => withdrawRemoval(token, e.id, r.id))}>
                Withdraw removal
              </button>
            </>
          ))}
      </div>
    </div>
  );
}

function StartRemovalDialog({ board, seat, onClose }: { board: Board; seat: Seat; onClose: () => void }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const [reason, setReason] = useState("");
  const reasonId = useId();
  const hasChair = board.seats.some((s) => s.position === "chair");
  const terminating = seat.position === "chair" || hasChair;
  const save = useMutation({
    mutationFn: () => startRemoval(session!.access_token, board.event.id, seat.id, reason),
    onSuccess: (b) => {
      store(b);
      onClose();
    },
  });
  useEffect(() => {
    const onKey = (ev: KeyboardEvent) => ev.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  const fake: Removal = {
    id: "", person: seat.id, name: seat.name, position: seat.position, reason: "", started_by: "",
    needs_vote: !terminating || seat.position === "chair", needs_chair: terminating && seat.position !== "chair",
    chair_agreed: false, leadership_approved: false, votes: [], waiting: [], my_vote: null, voting_closed: false,
    can_vote: false, can_close_voting: false, can_approve: false, can_agree: false, can_edit: false,
  };
  return (
    <div className="pe-scrim pe-overlay" onClick={(ev) => ev.target === ev.currentTarget && onClose()}>
      <div className="pe-dialog" role="dialog" aria-modal="true" aria-labelledby="removal-title">
        <h1 className="pe-dialog__title" id="removal-title">
          {terminating ? "Terminate" : "Start removal"}
        </h1>
        <p className="pe-dialog__body">
          {seat.name}, {POSITION[seat.position]} of {board.event.name}.
        </p>
        <p className="pe-note">{needs(board, fake)}</p>
        {save.isError && <Alert>{message(save.error)}</Alert>}
        <div className="pe-field">
          <label className="pe-field__label" htmlFor={reasonId}>
            Reason (optional)
          </label>
          <textarea className="pe-input" id={reasonId} value={reason} onChange={(ev) => setReason(ev.target.value)} />
        </div>
        <div className="pe-dialog__actions">
          <button className="pe-btn pe-btn--secondary" type="button" onClick={onClose}>
            Cancel
          </button>
          <button className="pe-btn pe-btn--danger" type="button" disabled={save.isPending} onClick={() => save.mutate()}>
            Start removal
          </button>
        </div>
      </div>
    </div>
  );
}

/* 7B.8: the Event Board elects the Deputy Board Chair from among the Board
   Members; the Event Chair records it, or Organization Leadership before an
   Event Chair exists. The name "Deputy Board Chair" is subject to change. */
function Deputy({ board }: { board: Board }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const members = board.seats.filter((s) => s.position === "member");
  const deputy = members.find((s) => s.deputy);
  const [person, setPerson] = useState("");
  const id = useId();
  const save = useMutation({
    mutationFn: () => recordDeputy(session!.access_token, board.event.id, person),
    onSuccess: (b) => {
      store(b);
      setPerson("");
    },
  });
  return (
    <section className="pe-card pe-stack" aria-labelledby="deputy-heading">
      <div className="pe-info-line">
        <h2 className="pe-card__title" id="deputy-heading">
          Deputy Board Chair
        </h2>
        <InfoButton label="About the Deputy Board Chair">
          The Deputy Board Chair serves in the place of the Event Chair when the Event Chair cannot serve. The Event
          Board elects the Deputy from among the Board Members; electing the Deputy is the first order of business at
          the board's initial meeting.
        </InfoButton>
      </div>
      {deputy ? (
        <p className="pe-dialog__body">
          <strong>{deputy.name}</strong> is the Deputy Board Chair.
        </p>
      ) : (
        <div>
          <Status>Not yet elected</Status>
        </div>
      )}
      {board.can_record_deputy && (
        <>
          {save.isError && <Alert>{message(save.error)}</Alert>}
          <div className="pe-field">
            <label className="pe-field__label" htmlFor={id}>
              The board's election
            </label>
            <select className="pe-input" id={id} value={person} onChange={(e) => setPerson(e.target.value)}>
              <option value="">Choose a Board Member</option>
              {members.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <button className="pe-btn pe-btn--secondary" type="button" disabled={!person || save.isPending} onClick={() => save.mutate()}>
              Record the board's election
            </button>
          </div>
        </>
      )}
    </section>
  );
}

/* An information icon (an "i" in a circle) that opens its text in a popup
   under it. Clicking it again, clicking elsewhere or pressing Esc closes it
   (owner, November 26, 2026). */
function InfoButton({ label, children }: { label: string; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);
  const popupId = useId();
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    const onClick = (e: MouseEvent) => ref.current && !ref.current.contains(e.target as Node) && setOpen(false);
    window.addEventListener("keydown", onKey);
    document.addEventListener("mousedown", onClick);
    return () => {
      window.removeEventListener("keydown", onKey);
      document.removeEventListener("mousedown", onClick);
    };
  }, [open]);
  return (
    <span className="pe-info" ref={ref}>
      <button
        className="pe-info__button"
        type="button"
        aria-label={label}
        aria-expanded={open}
        aria-controls={popupId}
        onClick={() => setOpen((o) => !o)}
      >
        i
      </button>
      {open && (
        <span className="pe-info__popup" id={popupId} role="note">
          {children}
        </span>
      )}
    </span>
  );
}

const WHY: Record<ReplaceWhy, string> = {
  resigned: "The chair resigned",
  illness: "An illness leaves the chair unable to resign",
  death: "The chair died",
};

/* D2, Replace chair, for the Event Board Chair (5D.3, owner's rulings of
   November 18, 2026). The position is offered outside PhoenixEvent:
   Organization Leadership records each Board Member's answer here. Board
   Members have first right of refusal, event participants second (they come
   with registration), the organization at large third. When several accept,
   Organization Leadership chooses. */
function ReplaceChairDialog({ board, chair, onClose }: { board: Board; chair: Seat; onClose: () => void }) {
  const { session } = useSession();
  const store = useBoardAction(board.event.id);
  const e = board.event;
  const members = board.seats.filter(
    (s) => s.position === "member" || (s.position === "event_administrator" && board.event_admin_holds_seat),
  );
  const [why, setWhy] = useState<ReplaceWhy | "">("");
  const [answers, setAnswers] = useState<Record<string, boolean>>({});
  const [person, setPerson] = useState("");
  const whyId = useId();
  const personId = useId();
  const accepted = members.filter((m) => answers[m.id] === true);
  const allAnswered = members.every((m) => m.id in answers);
  const tier: "board" | "organization" | "waiting" = accepted.length > 0 ? "board" : allAnswered ? "organization" : "waiting";
  const choices = tier === "board" ? accepted : tier === "organization" ? board.candidates : [];
  const full = tier === "organization" && why === "resigned" && board.seats_taken >= e.board_capacity;
  const save = useMutation({
    mutationFn: () => replaceChair(session!.access_token, e.id, { why: why as ReplaceWhy, answers, person }),
    onSuccess: (b) => {
      store(b);
      onClose();
    },
  });
  useEffect(() => {
    const onKey = (ev: KeyboardEvent) => ev.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  const answer = (id: string, value: string) => {
    setPerson("");
    setAnswers((a) => {
      const next = { ...a };
      if (value === "") delete next[id];
      else next[id] = value === "accepted";
      return next;
    });
  };
  const first = chair.name.split(" ")[0];
  return (
    <div className="pe-scrim pe-overlay" onClick={(ev) => ev.target === ev.currentTarget && onClose()}>
      <div className="pe-dialog" role="dialog" aria-modal="true" aria-labelledby="replace-title">
        <h1 className="pe-dialog__title" id="replace-title">
          Replace chair
        </h1>
        <p className="pe-dialog__body">
          {e.name}. Current chair: {chair.name}, Event Chair.
        </p>
        {save.isError && <Alert>{message(save.error)}</Alert>}
        <div className="pe-field">
          <label className="pe-field__label" htmlFor={whyId}>
            Why the chair is replaced
          </label>
          <select className="pe-input" id={whyId} value={why} onChange={(ev) => setWhy(ev.target.value as ReplaceWhy | "")}>
            <option value="">Choose why</option>
            {(Object.keys(WHY) as ReplaceWhy[]).map((w) => (
              <option key={w} value={w}>
                {WHY[w]}
              </option>
            ))}
          </select>
          <span className="pe-field__hint">A chair who is terminated is removed through Start removal.</span>
        </div>
        {members.length > 0 && (
          <fieldset className="pe-field">
            <legend className="pe-field__label">Board Members' answers (first right of refusal)</legend>
            {members.map((m) => (
              <div key={m.id} className="pe-list__item">
                <label className="pe-list__name" htmlFor={`answer-${m.id}`}>
                  {m.name}
                </label>
                <select
                  className="pe-input"
                  id={`answer-${m.id}`}
                  value={m.id in answers ? (answers[m.id] ? "accepted" : "declined") : ""}
                  onChange={(ev) => answer(m.id, ev.target.value)}
                >
                  <option value="">No answer yet</option>
                  <option value="accepted">Accepted</option>
                  <option value="declined">Declined</option>
                </select>
              </div>
            ))}
          </fieldset>
        )}
        <div className="pe-field">
          <label className="pe-field__label" htmlFor={personId}>
            {tier === "board" ? "New chair, from the Board Members who accepted" : "New chair, from the organization at large"}
          </label>
          <select className="pe-input" id={personId} value={person} disabled={tier === "waiting"} onChange={(ev) => setPerson(ev.target.value)}>
            <option value="">Choose the new chair</option>
            {choices.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <span className="pe-field__hint">
            {tier === "waiting"
              ? "Record every Board Member's answer first."
              : tier === "organization"
                ? `Every Board Member declined. Event participants come next; ${e.name} has none yet, since registration is not built, so the position goes to the organization at large.`
                : "Organization Leadership chooses among those who accepted."}
          </span>
        </div>
        {why && (
          <p className="pe-dialog__body">
            {why === "resigned"
              ? `${first} stays on the board as a Board Member and remains a participant of ${e.name}.`
              : why === "illness"
                ? `${first} leaves the board and stops being a participant of ${e.name}.`
                : `${first} leaves the board.`}
          </p>
        )}
        {full && (
          <Notice>
            Every seat on the board is taken. The Event Board decides what happens when the board is full; by default the
            board's size does not change.
          </Notice>
        )}
        <div className="pe-dialog__actions">
          <button className="pe-btn pe-btn--secondary" type="button" onClick={onClose}>
            Cancel
          </button>
          <button
            className="pe-btn pe-btn--primary"
            type="button"
            disabled={!why || !person || full || save.isPending}
            onClick={() => save.mutate()}
          >
            Replace chair
          </button>
        </div>
      </div>
    </div>
  );
}
