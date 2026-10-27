import { render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import App from "@/App";
import { SessionProvider } from "@/session";

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
