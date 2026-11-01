import { render } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import App from "@/App";
import { SessionProvider } from "@/session";
import { server } from "./server";

/* The whole web client at one path, as main.tsx puts it together. */
export function renderAt(path: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <MemoryRouter initialEntries={[path]}>
          <App />
        </MemoryRouter>
      </SessionProvider>
    </QueryClientProvider>,
  );
}

export const DANA = {
  id: "a",
  email: "dana@example.org",
  username: "dreyes",
  first_name: null,
  middle_name: null,
  last_name: null,
  name_shown: "username",
  name_choice: null,
  name: "dreyes",
  email_level: "private",
  username_level: "organization",
  name_level: "organization",
  roles_level: "organization",
  roles: [],
};

export const SESSION = {
  access_token: "t",
  expires_at: new Date(Date.now() + 15 * 60_000).toISOString(),
  account: { id: "a", email: "dana@example.org", username: "dreyes" },
};

/* Dana, logged in to Lincoln Elementary PTA, at one path. She holds no
   organization role unless a handler for /me says otherwise. */
export function signedInAt(path: string, ...handlers: Parameters<typeof server.use>) {
  server.use(
    http.get("/api/v1/address", () =>
      HttpResponse.json({
        kind: "organization",
        organization: { name: "Lincoln Elementary PTA", slug: "lincolnpta" },
        general_url: "http://phoenixevent.localhost:5174/",
      }),
    ),
    http.post("/api/v1/auth/refresh", () => HttpResponse.json(SESSION)),
    ...handlers,
    http.get("/api/v1/me", () => HttpResponse.json({ account: SESSION.account, organization: {}, roles: [] })),
  );
  renderAt(path);
}

