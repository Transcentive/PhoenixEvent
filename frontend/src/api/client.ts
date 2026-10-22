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
