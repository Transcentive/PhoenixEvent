import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { DANA, SESSION, signedInAt } from "./app";

const profile = http.get("/api/v1/profile", () => HttpResponse.json(DANA));
const administrator = http.get("/api/v1/me", () =>
  HttpResponse.json({ account: SESSION.account, organization: {}, roles: ["organization_administrator"] }),
);
const POLICIES = { approve_registrations: true, participation: "both", event_admin_holds_seat: false };

describe("O1, events", () => {
  it("shows a member the events, private, with no Create event or Policies", async () => {
    signedInAt(
      "/",
      profile,
      http.get("/api/v1/events", () =>
        HttpResponse.json({
          events: [
            { id: "e1", name: "Halloween Bash", date: "2026-10-31", location: "Gym", description: null, board_capacity: 5, visibility: "private", board_order: "board_first", can_edit: false, formed: false },
          ],
        }),
      ),
    );
    expect(await screen.findByRole("heading", { name: "Halloween Bash" })).toBeInTheDocument();
    expect(screen.getByText("October 31, 2026 · Gym")).toBeInTheDocument();
    expect(screen.getByText("Private")).toBeInTheDocument();
    expect(screen.getByText("Event board forming · 0 stations")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Create event" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Policies" })).not.toBeInTheDocument();
  });

  it("shows an Organization Administrator Create event and Policies", async () => {
    signedInAt("/", profile, administrator);
    expect(await screen.findByRole("link", { name: "Create event" })).toHaveAttribute("href", "/events/new");
    expect(screen.getByRole("link", { name: "Policies" })).toHaveAttribute("href", "/policies");
  });
});

describe("O3, Policies", () => {
  it("shows all four sections, the later two marked and not changeable, and saves", async () => {
    let sent: unknown = null;
    signedInAt(
      "/policies",
      profile,
      administrator,
      http.get("/api/v1/policies", () => HttpResponse.json(POLICIES)),
      http.put("/api/v1/policies", async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>;
        sent = body;
        return HttpResponse.json(body);
      }),
    );
    expect(await screen.findByRole("heading", { name: "Registration" })).toBeInTheDocument();
    for (const name of ["Waiting lists", "Event Administrator", "Lowering a station's capacity"]) {
      expect(screen.getByRole("heading", { name })).toBeInTheDocument();
    }
    expect(screen.getAllByText(/This setting comes in a later release/)).toHaveLength(2);
    expect(screen.getByRole("radio", { name: /Leave it to the event board/ })).toBeDisabled();
    expect(screen.getByRole("radio", { name: /^Refuse/ })).toBeDisabled();
    expect(screen.getByRole("radio", { name: /^Both/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /^Holds no seat/ })).toBeChecked();

    await userEvent.click(screen.getByRole("checkbox", { name: /Approve each registration/ }));
    await userEvent.click(screen.getByRole("radio", { name: /^Subscribers/ }));
    await userEvent.click(screen.getByRole("radio", { name: /^Holds a seat/ }));
    await userEvent.click(screen.getByRole("button", { name: "Save policies" }));
    expect(await screen.findByText("Policies saved.")).toBeInTheDocument();
    expect(sent).toEqual({ approve_registrations: false, participation: "subscribers", event_admin_holds_seat: true });
  });

  it("is outside a member's access", async () => {
    signedInAt(
      "/policies",
      profile,
      http.get("/api/v1/policies", () =>
        HttpResponse.json({ code: "no_access", message: "This page is outside your access." }, { status: 404 }),
      ),
    );
    expect(await screen.findByText("This page is outside your access")).toBeInTheDocument();
  });
});

describe("S1, Create event", () => {
  it("shows field errors, then creates the event and returns to O1", async () => {
    let sent: Record<string, unknown> | null = null;
    let created = false;
    signedInAt(
      "/events/new",
      profile,
      administrator,
      http.post("/api/v1/events", async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>;
        if (!body.name) {
          return HttpResponse.json(
            {
              code: "invalid_fields",
              message: "Check the fields marked below.",
              fields: [
                { field: "name", message: "Enter the event's name." },
                { field: "board_order", message: "Choose which comes first, the board or the chair." },
              ],
            },
            { status: 422 },
          );
        }
        sent = body;
        created = true;
        return HttpResponse.json({ id: "e1", ...body, visibility: "private" }, { status: 201 });
      }),
      http.get("/api/v1/events", () =>
        HttpResponse.json({
          events: created
            ? [{ id: "e1", name: "Spring Fair", date: null, location: null, description: null, board_capacity: 5, visibility: "private", board_order: "chair_first", can_edit: true, formed: false }]
            : [],
        }),
      ),
    );
    expect(await screen.findByRole("heading", { level: 1, name: "Create event" })).toBeInTheDocument();
    expect(screen.getByText("Private. Members of Lincoln Elementary PTA see it after logging in.")).toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /^Public/ })).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Create event" }));
    expect(await screen.findByText("Enter the event's name.")).toBeInTheDocument();
    expect(screen.getByText("Choose which comes first, the board or the chair.")).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText("Event name"), "Spring Fair");
    await userEvent.type(screen.getByLabelText("Event board capacity"), "5");
    await userEvent.click(screen.getByRole("radio", { name: /^The chair first/ }));
    await userEvent.click(screen.getByRole("button", { name: "Create event" }));
    expect(await screen.findByRole("heading", { name: "Spring Fair" })).toBeInTheDocument();
    expect(sent).toMatchObject({ name: "Spring Fair", board_capacity: "5", board_order: "chair_first", date: "", location: "" });
  });

  it("is outside a member's access", async () => {
    signedInAt("/events/new", profile);
    expect(await screen.findByText("This page is outside your access")).toBeInTheDocument();
    expect(within(document.body).queryByRole("button", { name: "Create event" })).not.toBeInTheDocument();
  });
});

describe("Edit event", () => {
  const EVENT = { id: "e1", name: "Turkey Triathlon", date: "2026-11-28", location: "High School Parking Lot", description: null, board_capacity: 5, visibility: "private", board_order: "board_first", can_edit: true, formed: false };

  it("shows Edit event on the card to those who may edit, and saves changes", async () => {
    let sent: Record<string, unknown> | null = null;
    signedInAt(
      "/",
      profile,
      administrator,
      http.get("/api/v1/events", () => HttpResponse.json({ events: [sent ? { ...EVENT, name: "Turkey Trot" } : EVENT] })),
      http.get("/api/v1/events/e1", () => HttpResponse.json(EVENT)),
      http.put("/api/v1/events/e1", async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>;
        sent = body;
        return HttpResponse.json({ ...EVENT, name: body.name });
      }),
    );
    await userEvent.click(await screen.findByRole("link", { name: "Edit event" }));
    expect(await screen.findByRole("heading", { level: 1, name: "Edit event" })).toBeInTheDocument();
    expect(screen.getByLabelText("Event name")).toHaveValue("Turkey Triathlon");
    expect(screen.getByLabelText("Event board capacity")).toHaveValue("5");
    // The order shows but cannot change (5E.4).
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    expect(screen.getByText(/^The board first/)).toBeInTheDocument();
    await userEvent.clear(screen.getByLabelText("Event name"));
    await userEvent.type(screen.getByLabelText("Event name"), "Turkey Trot");
    await userEvent.click(screen.getByRole("button", { name: "Save event" }));
    expect(await screen.findByRole("heading", { name: "Turkey Trot" })).toBeInTheDocument();
    expect(sent).toMatchObject({ name: "Turkey Trot", date: "2026-11-28", board_capacity: "5", board_order: "board_first" });
  });

  it("is outside the access of someone who may not edit", async () => {
    signedInAt(
      "/events/e1/edit",
      profile,
      http.get("/api/v1/events/e1", () =>
        HttpResponse.json({ code: "no_access", message: "This page is outside your access." }, { status: 404 }),
      ),
    );
    expect(await screen.findByText("This page is outside your access")).toBeInTheDocument();
  });
});

describe("Event board", () => {
  const EVENT = { id: "e1", name: "Valentine Dance", date: null, location: null, description: null, board_capacity: 5, visibility: "private", board_order: "board_first", can_edit: false, formed: false };
  const BOARD = {
    event: EVENT,
    seats: [{ id: "m1", name: "Lee Park", position: "member", deputy: false }],
    seats_taken: 1,
    event_admin_holds_seat: false,
    formed: false,
    candidates: [{ id: "c1", name: "Sam Ortiz" }],
    can_appoint_chair: false,
    can_record_chair: true,
    can_appoint_members: true,
    can_appoint_admin: true,
    can_remove: false,
    can_record_deputy: false,
    can_start_removal: [],
    removals: [],
  };

  it("forms a board-first board: appoints in D1 and records the chair", async () => {
    let appointed: unknown = null;
    let recorded: unknown = null;
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () => HttpResponse.json(BOARD)),
      http.post("/api/v1/events/e1/board", async ({ request }) => {
        appointed = await request.json();
        return HttpResponse.json({
          ...BOARD,
          seats: [...BOARD.seats, { id: "c1", name: "Sam Ortiz", position: "member", deputy: false }],
          seats_taken: 2,
          candidates: [],
        });
      }),
      http.post("/api/v1/events/e1/board/chair", async ({ request }) => {
        recorded = await request.json();
        return HttpResponse.json({
          ...BOARD,
          seats: [{ id: "m1", name: "Lee Park", position: "chair", deputy: false }],
          formed: false,
          can_record_chair: false,
        });
      }),
    );
    expect(await screen.findByRole("heading", { level: 1, name: "Event board" })).toBeInTheDocument();
    expect(screen.getByText(/forms its board first/)).toBeInTheDocument();
    expect(screen.getByText("1 of 5 seats taken")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Appoint board member" }));
    const dialog = await screen.findByRole("dialog", { name: "Appoint board member" });
    await userEvent.selectOptions(within(dialog).getByLabelText("Member"), "c1");
    expect(within(dialog).getByRole("option", { name: "Event Administrator" })).toBeInTheDocument();
    await userEvent.click(within(dialog).getByRole("button", { name: "Appoint board member" }));
    expect(await screen.findByText("Sam Ortiz", { selector: ".pe-list__name" })).toBeInTheDocument();
    expect(appointed).toEqual({ person: "c1", position: "member" });

    await userEvent.selectOptions(screen.getByLabelText("The board's choice"), "m1");
    await userEvent.click(screen.getByRole("button", { name: "Record the board's choice" }));
    expect(await screen.findByText(/is the Event Chair/)).toBeInTheDocument();
    expect(recorded).toEqual({ person: "m1" });
  });

  it("lists all three roles in D1, greying out those the person may not appoint", async () => {
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () =>
        HttpResponse.json({
          ...BOARD,
          event: { ...EVENT, board_order: "chair_first" },
          seats: [{ id: "m1", name: "Lee Park", position: "chair", deputy: false }],
          can_record_chair: false,
          can_appoint_members: false,
        }),
      ),
    );
    await userEvent.click(await screen.findByRole("button", { name: "Appoint board member" }));
    const dialog = await screen.findByRole("dialog", { name: "Appoint board member" });
    expect(within(dialog).getByRole("option", { name: "Event Chair (the seat is filled)" })).toBeDisabled();
    expect(within(dialog).getByRole("option", { name: "Board Member (only the Event Chair appoints Board Members)" })).toBeDisabled();
    const admin = within(dialog).getByRole("option", { name: "Event Administrator" }) as HTMLOptionElement;
    expect(admin).toBeEnabled();
    expect(admin.selected).toBe(true);
  });

  it("replaces the chair in D2: Board Members first, then the organization at large", async () => {
    let sent: unknown = null;
    const withChair = {
      ...BOARD,
      event: { ...EVENT, board_order: "chair_first" },
      seats: [
        { id: "d1", name: "Dana Reyes", position: "chair", deputy: false },
        { id: "m1", name: "Lee Park", position: "member", deputy: false },
      ],
      seats_taken: 2,
      can_record_chair: false,
      can_replace_chair: true,
    };
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () => HttpResponse.json(withChair)),
      http.post("/api/v1/events/e1/board/replace-chair", async ({ request }) => {
        sent = await request.json();
        return HttpResponse.json({
          ...withChair,
          seats: [
            { id: "c1", name: "Sam Ortiz", position: "chair", deputy: false },
            { id: "d1", name: "Dana Reyes", position: "member", deputy: false },
            { id: "m1", name: "Lee Park", position: "member", deputy: false },
          ],
        });
      }),
    );
    await userEvent.click(await screen.findByRole("button", { name: "Replace chair" }));
    const dialog = await screen.findByRole("dialog", { name: "Replace chair" });
    const newChair = within(dialog).getByLabelText("New chair, from the organization at large");
    expect(newChair).toBeDisabled();
    expect(within(dialog).getByText("Record every Board Member's answer first.")).toBeInTheDocument();

    await userEvent.selectOptions(within(dialog).getByLabelText("Why the chair is replaced"), "resigned");
    expect(within(dialog).getByText(/Dana stays on the board as a Board Member/)).toBeInTheDocument();
    await userEvent.selectOptions(within(dialog).getByLabelText("Lee Park"), "declined");
    expect(within(dialog).getByText(/Every Board Member declined/)).toBeInTheDocument();
    await userEvent.selectOptions(within(dialog).getByLabelText("New chair, from the organization at large"), "c1");
    await userEvent.click(within(dialog).getByRole("button", { name: "Replace chair" }));
    expect(await screen.findByText(/Sam Ortiz/, { selector: "strong" })).toBeInTheDocument();
    expect(sent).toEqual({ why: "resigned", answers: { m1: false }, person: "c1" });
  });

  it("shows a member the board with no actions", async () => {
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () =>
        HttpResponse.json({ ...BOARD, can_record_chair: false, can_appoint_members: false, can_appoint_admin: false, can_remove: false }),
      ),
    );
    expect(await screen.findByText("Lee Park")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Appoint board member" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("The board's choice")).not.toBeInTheDocument();
  });

  it("tells someone without Organization Leadership why there is nothing to do", async () => {
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () =>
        HttpResponse.json({ ...BOARD, event: { ...EVENT, board_order: "chair_first" }, seats: [], seats_taken: 0, can_appoint_chair: false, can_record_chair: false, can_appoint_members: false, can_appoint_admin: false, can_remove: false }),
      ),
    );
    expect(await screen.findByText(/You need Organization Leadership to appoint the chair/)).toBeInTheDocument();
    expect(screen.getByText(/can give you that role with Change roles on your member profile/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Appoint chair" })).not.toBeInTheDocument();
  });

  it("appoints the chair first on a chair-first board", async () => {
    let appointed: unknown = null;
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () =>
        HttpResponse.json({ ...BOARD, event: { ...EVENT, board_order: "chair_first" }, seats: [], seats_taken: 0, can_appoint_chair: true, can_record_chair: false, can_appoint_members: false }),
      ),
      http.post("/api/v1/events/e1/board", async ({ request }) => {
        appointed = await request.json();
        return HttpResponse.json({ ...BOARD, event: { ...EVENT, board_order: "chair_first" }, seats: [{ id: "c1", name: "Sam Ortiz", position: "chair", deputy: false }] });
      }),
    );
    await userEvent.selectOptions(await screen.findByLabelText("Member to appoint as chair"), "c1");
    await userEvent.click(screen.getByRole("button", { name: "Appoint chair" }));
    expect(await screen.findByText(/is the Event Chair/)).toBeInTheDocument();
    expect(appointed).toEqual({ person: "c1", position: "chair" });
  });
});

describe("Delete event", () => {
  it("deletes after confirming, and returns to the organization home", async () => {
    let deleted = false;
    const EVENT = { id: "e1", name: "Turkey Triathlon", date: null, location: null, description: null, board_capacity: 5, visibility: "private", board_order: "board_first", can_edit: true, formed: false };
    signedInAt(
      "/events/e1/edit",
      profile,
      http.get("/api/v1/events/e1", () => HttpResponse.json(EVENT)),
      http.delete("/api/v1/events/e1", () => {
        deleted = true;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    await userEvent.click(await screen.findByRole("button", { name: "Delete event" }));
    const confirm = screen.getByRole("alertdialog", { name: "Delete Turkey Triathlon?" });
    await userEvent.click(within(confirm).getByRole("button", { name: "Delete event" }));
    expect(await screen.findByText("No events yet")).toBeInTheDocument();
    expect(deleted).toBe(true);
  });
});

describe("Removals from the event board", () => {
  const EVENT = { id: "e1", name: "Valentine Dance", date: null, location: null, description: null, board_capacity: 5, visibility: "private", board_order: "board_first", can_edit: false, formed: false };
  const BOARD = {
    event: EVENT,
    seats: [
      { id: "m1", name: "Lee Park", position: "member", deputy: false },
      { id: "m2", name: "Sam Ortiz", position: "member", deputy: false },
    ],
    seats_taken: 2,
    event_admin_holds_seat: false,
    formed: false,
    candidates: [],
    can_appoint_chair: false,
    can_record_chair: false,
    can_appoint_members: false,
    can_appoint_admin: false,
    can_remove: false,
    can_record_deputy: false,
    can_start_removal: ["m2"],
    removals: [],
  };
  const REMOVAL = {
    id: "r1", person: "m2", name: "Sam Ortiz", position: "member", reason: "", started_by: "Lee Park",
    needs_vote: true, needs_chair: false, chair_agreed: false, leadership_approved: false,
    votes: [{ name: "Lee Park", in_favor: true }], waiting: ["Dana Reyes"], my_vote: null, voting_closed: false,
    can_vote: true, can_close_voting: false, can_approve: false, can_agree: false, can_edit: false,
  };

  it("starts a removal before the chair exists, then votes on it", async () => {
    let started: unknown = null;
    let voted: unknown = null;
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () => HttpResponse.json(BOARD)),
      http.post("/api/v1/events/e1/removals", async ({ request }) => {
        started = await request.json();
        return HttpResponse.json({ ...BOARD, removals: [REMOVAL] });
      }),
      http.post("/api/v1/events/e1/removals/r1/vote", async ({ request }) => {
        voted = await request.json();
        return HttpResponse.json(BOARD);
      }),
    );
    await userEvent.click(await screen.findByRole("button", { name: "Start removal" }));
    const dialog = await screen.findByRole("dialog", { name: "Start removal" });
    expect(within(dialog).getByText(/unanimous vote of the Event Board, that member recused/)).toBeInTheDocument();
    await userEvent.click(within(dialog).getByRole("button", { name: "Start removal" }));
    expect(started).toEqual({ person: "m2", reason: "" });
    expect(await screen.findByRole("heading", { name: "Removals" })).toBeInTheDocument();
    expect(screen.getByText(/abstaining at the close\): Dana Reyes/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Vote in favor" }));
    expect(voted).toEqual({ in_favor: true });
  });

  it("changes a vote and closes voting", async () => {
    let closed: unknown = null;
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () =>
        HttpResponse.json({ ...BOARD, removals: [{ ...REMOVAL, my_vote: true, can_close_voting: true }] }),
      ),
      http.post("/api/v1/events/e1/removals/r1/close", async ({ request }) => {
        closed = await request.json();
        return HttpResponse.json(BOARD);
      }),
    );
    expect(await screen.findByRole("button", { name: "Change vote to against" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Vote in favor" })).not.toBeInTheDocument();
    expect(screen.getByText(/Not voted \(abstaining at the close\): Dana Reyes/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Close voting: carries" }));
    await waitFor(() => expect(closed).toEqual({ carries: true }));
  });

  it("terminates the Event Chair with an optional reason", async () => {
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () =>
        HttpResponse.json({ ...BOARD, seats: [{ id: "c1", name: "Dana Reyes", position: "chair", deputy: false }, ...BOARD.seats], can_start_removal: ["c1"] }),
      ),
    );
    await userEvent.click(await screen.findByRole("button", { name: "Start removal" }));
    const dialog = await screen.findByRole("dialog", { name: "Terminate" });
    expect(within(dialog).getByRole("button", { name: "Start removal" })).toBeEnabled();
    await userEvent.type(within(dialog).getByLabelText("Reason (optional)"), "Missed every meeting");
    expect(within(dialog).getByRole("button", { name: "Start removal" })).toBeEnabled();
  });

  it("records the board's election of the Deputy Board Chair", async () => {
    let sent: unknown = null;
    signedInAt(
      "/events/e1/board",
      profile,
      http.get("/api/v1/events/e1/board", () => HttpResponse.json({ ...BOARD, can_record_deputy: true })),
      http.post("/api/v1/events/e1/board/deputy", async ({ request }) => {
        sent = await request.json();
        return HttpResponse.json({ ...BOARD, seats: [BOARD.seats[0], { ...BOARD.seats[1], deputy: true }] });
      }),
    );
    expect(await screen.findByText("Not yet elected")).toBeInTheDocument();
    expect(screen.queryByText(/serves in the place of the Event Chair/)).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "About the Deputy Board Chair" }));
    expect(screen.getByText(/serves in the place of the Event Chair/)).toBeInTheDocument();
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByText(/serves in the place of the Event Chair/)).not.toBeInTheDocument();
    await userEvent.selectOptions(screen.getByLabelText("The board's election"), "m2");
    await userEvent.click(screen.getByRole("button", { name: "Record the board's election" }));
    expect(await screen.findByText(/is the Deputy Board Chair\./)).toBeInTheDocument();
    expect(sent).toEqual({ person: "m2" });
  });
});
