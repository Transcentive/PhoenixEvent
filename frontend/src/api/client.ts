/* The API is always same-origin: Vite proxies /api/v1 in development and
   nginx does on the droplet, so no address is configured here. */

export interface Health {
  status: "ok" | "unavailable";
  database: "ok" | "unreachable";
  build: string;
}

export async function getHealth(): Promise<Health> {
  const response = await fetch("/api/v1/health", {
    headers: { Accept: "application/json" },
  });
  /* 503 carries the same body as 200, saying what is unavailable. Anything
     else means the API itself did not answer. */
  if (response.ok || response.status === 503) {
    return (await response.json()) as Health;
  }
  throw new Error(`The API answered ${response.status}`);
}

export interface Organization {
  name: string;
  slug: string;
}

export interface Account {
  id: string;
  email: string;
  username: string | null;
}

export interface Session {
  access_token: string;
  expires_at: string;
  account: Account;
}

export interface LoggedInElsewhere {
  name: string;
  logout_url: string;
}

export type Address =
  | { kind: "general" }
  | {
      kind: "organization";
      organization: Organization;
      general_url: string;
      logged_in_elsewhere?: LoggedInElsewhere;
    }
  | { kind: "unknown" };

export interface FieldError {
  field: string;
  message: string;
}

/* Every refusal carries a code and a message written for the person. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly fields: FieldError[] = [],
    readonly loggedInElsewhere?: LoggedInElsewhere,
  ) {
    super(message);
  }
}

async function call<T>(method: string, path: string, body?: unknown, accessToken?: string): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
  const response = await fetch(`/api/v1${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: "same-origin",
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new ApiError(
      response.status,
      data.code ?? "unknown",
      data.message ?? "Something went wrong on our side. Try again.",
      data.fields ?? [],
      data.logged_in_elsewhere,
    );
  }
  return data as T;
}

export async function getAddress(): Promise<Address> {
  try {
    return await call<Address>("GET", "/address");
  } catch (e) {
    if (e instanceof ApiError && e.code === "unknown_address") return { kind: "unknown" };
    throw e;
  }
}

export interface OrganizationMatch extends Organization {
  login_url: string;
}

export const matchOrganizations = (typed: string, signal?: AbortSignal) =>
  fetch(`/api/v1/organizations?q=${encodeURIComponent(typed)}`, { headers: { Accept: "application/json" }, signal })
    .then((r) => (r.ok ? r.json() : { organizations: [] }))
    .then((d: { organizations: OrganizationMatch[] }) => d.organizations);

export const findOrganization = (email: string) =>
  call<{ message: string }>("POST", "/find-organization", { email });

export const register = (email: string, username: string, password: string) =>
  call<{ email: string }>("POST", "/auth/register", { email, username, password });

export const resendConfirmation = (email: string) =>
  call<object>("POST", "/auth/resend-confirmation", { email });

export const confirmEmail = (token: string) => call<object>("POST", "/auth/confirm-email", { token });

export const linkPurpose = (token: string) =>
  call<{ purpose: string; login: string }>("GET", `/auth/link?token=${encodeURIComponent(token)}`);

export const forgotPassword = (login: string) => call<object>("POST", "/auth/forgot-password", { login });

export const resetPassword = (token: string, password: string) =>
  call<object>("POST", "/auth/reset-password", { token, password });

export const login = (login: string, password: string) =>
  call<Session>("POST", "/auth/login", { login, password });

export const refresh = () => call<Session>("POST", "/auth/refresh");

export const logout = () => call<object>("POST", "/auth/logout");

export const getMe = (accessToken: string) =>
  call<{ account: Account; organization: Organization }>("GET", "/me", undefined, accessToken);

export type Level = "public" | "organization" | "event" | "station" | "private";
export type NameShown = "email" | "username" | "personal_name";

/* The owner's own profile (A8, spec 8B). */
export interface Profile {
  id: string;
  email: string;
  username: string | null;
  first_name: string | null;
  middle_name: string | null;
  last_name: string | null;
  name_shown: NameShown;
  name_choice: NameShown | null;
  name: string;
  email_level: Level;
  username_level: Level;
  name_level: Level;
  roles_level: Level;
  roles: string[];
}

export interface ProfileChange {
  username: string;
  first_name: string;
  middle_name: string;
  last_name: string;
}

export interface VisibilityChange {
  name_shown: NameShown | "";
  email_level: Level;
  username_level: Level;
  name_level: Level;
  roles_level: Level;
}

/* Another person's profile as the viewer may see it (A9). */
export interface SharedProfile {
  id: string;
  name: string;
  personal_name?: string;
  username?: string;
  email?: string;
  roles?: string[];
}

export const getProfile = (accessToken: string) => call<Profile>("GET", "/profile", undefined, accessToken);

export const updateProfile = (accessToken: string, change: ProfileChange) =>
  call<Profile>("PUT", "/profile", change, accessToken);

export const updateVisibility = (accessToken: string, change: VisibilityChange) =>
  call<Profile>("PUT", "/profile/visibility", change, accessToken);

export const changeEmail = (accessToken: string, email: string, password: string) =>
  call<Profile>("POST", "/profile/email", { email, password }, accessToken);

export const getPerson = (accessToken: string, id: string) =>
  call<SharedProfile>("GET", `/people/${encodeURIComponent(id)}`, undefined, accessToken);

/* Organization roles (7B). Member is every account and is not listed. */
export type OrganizationRole = "organization_administrator" | "organization_leadership";

export interface Officer {
  id: string;
  name: string;
  roles: string[];
}

export interface PersonRoles {
  id: string;
  name: string;
  roles: OrganizationRole[];
}

export const getMyRoles = (accessToken: string) =>
  call<{ roles: OrganizationRole[] }>("GET", "/me", undefined, accessToken).then((d) => d.roles);

export const getOfficers = (accessToken: string) =>
  call<{ officers: Officer[] }>("GET", "/officers", undefined, accessToken).then((d) => d.officers);

export const getPersonRoles = (accessToken: string, id: string) =>
  call<PersonRoles>("GET", `/people/${encodeURIComponent(id)}/roles`, undefined, accessToken);

export const setPersonRoles = (accessToken: string, id: string, roles: OrganizationRole[]) =>
  call<PersonRoles>("PUT", `/people/${encodeURIComponent(id)}/roles`, { roles }, accessToken);

/* O3, Policies (16B.6): the settings the first release acts on. */
export type Participation = "subscribers" | "registrants" | "both";

export interface Policies {
  approve_registrations: boolean;
  participation: Participation;
  event_admin_holds_seat: boolean;
}

export const getPolicies = (accessToken: string) => call<Policies>("GET", "/policies", undefined, accessToken);

export const setPolicies = (accessToken: string, p: Policies) => call<Policies>("PUT", "/policies", p, accessToken);

/* Events: O1 lists them, S1 creates one (16B.6). */
export type BoardOrder = "board_first" | "chair_first";

export interface EventSummary {
  id: string;
  name: string;
  date: string | null;
  location: string | null;
  description: string | null;
  board_capacity: number;
  visibility: "public" | "unlisted" | "private";
  board_order: BoardOrder;
  can_edit: boolean;
  formed: boolean;
}

export interface NewEvent {
  name: string;
  date: string;
  location: string;
  description: string;
  board_capacity: string;
  board_order: BoardOrder | "";
}

export const getEvents = (accessToken: string) =>
  call<{ events: EventSummary[] }>("GET", "/events", undefined, accessToken).then((d) => d.events);

export const createEvent = (accessToken: string, e: NewEvent) =>
  call<EventSummary>("POST", "/events", e, accessToken);

export const getEvent = (accessToken: string, id: string) =>
  call<EventSummary>("GET", `/events/${encodeURIComponent(id)}`, undefined, accessToken);

export const updateEvent = (accessToken: string, id: string, e: NewEvent) =>
  call<EventSummary>("PUT", `/events/${encodeURIComponent(id)}`, e, accessToken);

export const deleteEvent = (accessToken: string, id: string) =>
  call<object>("DELETE", `/events/${encodeURIComponent(id)}`, undefined, accessToken);

/* The event board: forming it board first or chair first (5E), and D1. */
export type Position = "chair" | "member" | "event_administrator";

export interface Seat {
  id: string;
  name: string;
  position: Position;
  deputy: boolean;
}

export interface Board {
  event: EventSummary;
  seats: Seat[];
  seats_taken: number;
  event_admin_holds_seat: boolean;
  formed: boolean;
  candidates: { id: string; name: string }[];
  can_appoint_chair: boolean;
  can_record_chair: boolean;
  can_appoint_members: boolean;
  can_appoint_admin: boolean;
  can_remove: boolean;
  can_record_deputy: boolean;
  can_replace_chair: boolean;
  can_start_removal: string[];
  removals: Removal[];
}

/* A removal from the event board (5D.3, rulings of November 11, 2026). */
export interface Removal {
  id: string;
  person: string;
  name: string;
  position: Position;
  reason: string;
  started_by: string;
  needs_vote: boolean;
  needs_chair: boolean;
  chair_agreed: boolean;
  leadership_approved: boolean;
  votes: { name: string; in_favor: boolean }[];
  my_vote: boolean | null;
  voting_closed: boolean;
  waiting: string[];
  can_vote: boolean;
  can_close_voting: boolean;
  can_approve: boolean;
  can_agree: boolean;
  can_edit: boolean;
}

const boardPath = (id: string) => `/events/${encodeURIComponent(id)}/board`;

export const getBoard = (accessToken: string, id: string) => call<Board>("GET", boardPath(id), undefined, accessToken);

export const appoint = (accessToken: string, id: string, person: string, position: Position) =>
  call<Board>("POST", boardPath(id), { person, position }, accessToken);

export const recordChair = (accessToken: string, id: string, person: string) =>
  call<Board>("POST", `${boardPath(id)}/chair`, { person }, accessToken);

/* D2, Replace chair, for the Event Board Chair (5D.3, rulings of November 18,
   2026): why, each Board Member's answer (true accepted), and the new chair. */
export type ReplaceWhy = "resigned" | "illness" | "death";

export const replaceChair = (
  accessToken: string,
  id: string,
  body: { why: ReplaceWhy; answers: Record<string, boolean>; person: string },
) => call<Board>("POST", `${boardPath(id)}/replace-chair`, body, accessToken);

/* Records the Event Board's election of the Deputy Board Chair (7B.8). */
export const recordDeputy = (accessToken: string, id: string, person: string) =>
  call<Board>("POST", `${boardPath(id)}/deputy`, { person }, accessToken);

export const removeFromBoard = (accessToken: string, id: string, person: string) =>
  call<Board>("DELETE", `${boardPath(id)}/${encodeURIComponent(person)}`, undefined, accessToken);

const removalsPath = (id: string) => `/events/${encodeURIComponent(id)}/removals`;

export const startRemoval = (accessToken: string, id: string, person: string, reason: string) =>
  call<Board>("POST", removalsPath(id), { person, reason }, accessToken);

export const editRemoval = (accessToken: string, id: string, removal: string, reason: string) =>
  call<Board>("PUT", `${removalsPath(id)}/${encodeURIComponent(removal)}`, { reason }, accessToken);

export const withdrawRemoval = (accessToken: string, id: string, removal: string) =>
  call<Board>("DELETE", `${removalsPath(id)}/${encodeURIComponent(removal)}`, undefined, accessToken);

export const voteOnRemoval = (accessToken: string, id: string, removal: string, inFavor: boolean) =>
  call<Board>("POST", `${removalsPath(id)}/${encodeURIComponent(removal)}/vote`, { in_favor: inFavor }, accessToken);

export const decideRemoval = (accessToken: string, id: string, removal: string, approve: boolean) =>
  call<Board>("POST", `${removalsPath(id)}/${encodeURIComponent(removal)}/decision`, { approve }, accessToken);

/* carries: the Event Board's decision when someone abstained. */
export const closeVoting = (accessToken: string, id: string, removal: string, carries?: boolean) =>
  call<Board>("POST", `${removalsPath(id)}/${encodeURIComponent(removal)}/close`, carries === undefined ? {} : { carries }, accessToken);
