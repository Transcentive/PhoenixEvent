import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { http, HttpResponse } from "msw";
import App from "@/App";
import { SessionProvider } from "@/session";
import { safeReturn } from "@/pages/account/Logout";
import { server } from "./server";

const PTA = {
  kind: "organization",
  organization: { name: "Lincoln Elementary PTA", slug: "lincolnpta" },
  general_url: "http://phoenixevent.localhost:5174/",
};

const ENDED = { code: "session_ended", message: "Your session has ended. Log in again." };

function renderAt(path: string) {
  render(
    <SessionProvider>
      <MemoryRouter initialEntries={[path]}>
        <App />
      </MemoryRouter>
    </SessionProvider>,
  );
}

function atPTA(...handlers: Parameters<typeof server.use>) {
  server.use(
    http.get("/api/v1/address", () => HttpResponse.json(PTA)),
    http.post("/api/v1/auth/refresh", () => HttpResponse.json(ENDED, { status: 401 })),
    ...handlers,
  );
}

describe("account screens", () => {
  it("A1 answers the lookup the same way, without a list", async () => {
    server.use(
      http.get("/api/v1/address", () => HttpResponse.json({ kind: "general" })),
      http.post("/api/v1/find-organization", () =>
        HttpResponse.json(
          { message: "If any organization has an account under this address, we've emailed you a link to each one." },
          { status: 202 },
        ),
      ),
    );
    renderAt("/");
    await userEvent.type(await screen.findByLabelText("Email address"), "dana@example.org");
    await userEvent.click(screen.getByRole("button", { name: "Email me the links" }));
    expect(await screen.findByText(/we've emailed you a link to each one/)).toBeInTheDocument();
  });

  it("A1's Organization field offers the organizations that match what is typed", async () => {
    const asked: string[] = [];
    server.use(
      http.get("/api/v1/address", () => HttpResponse.json({ kind: "general" })),
      http.get("/api/v1/organizations", ({ request }) => {
        asked.push(new URL(request.url).searchParams.get("q") ?? "");
        return HttpResponse.json({
          organizations: [
            {
              name: "Lincoln Elementary PTA",
              slug: "lincolnpta",
              login_url: "http://lincolnpta.phoenixevent.localhost:5174/login",
            },
          ],
        });
      }),
    );
    renderAt("/");
    await userEvent.type(await screen.findByRole("combobox", { name: "Organization" }), "lincoln");
    const option = await screen.findByRole("option", { name: /Lincoln Elementary PTA/ });
    expect(option).toHaveTextContent("lincolnpta");
    expect(asked.at(-1)).toBe("lincoln");
  });

  it("an organization's address shows its login page", async () => {
    atPTA();
    renderAt("/");
    expect(await screen.findByRole("heading", { name: "Log in" })).toBeInTheDocument();
    expect(screen.getByText(/Lincoln Elementary PTA/)).toBeInTheDocument();
  });

  it("E1: a failed login shows the message and clears the password", async () => {
    atPTA(
      http.post("/api/v1/auth/login", () =>
        HttpResponse.json(
          {
            code: "bad_credentials",
            message:
              "That username and password do not match an account in this organization. Check both and try again, or reset your password.",
          },
          { status: 401 },
        ),
      ),
    );
    renderAt("/login");
    await userEvent.type(await screen.findByLabelText("Username"), "dreyes");
    await userEvent.type(screen.getByLabelText("Password"), "wrong password here");
    await userEvent.click(screen.getByRole("button", { name: "Log in" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("do not match an account in this organization");
    expect(screen.getByLabelText("Password")).toHaveValue("");
    expect(screen.getByLabelText("Username")).toHaveValue("dreyes");
  });

  it("a login shows whom the session belongs to, and logging out shows A7", async () => {
    const session = {
      access_token: "t",
      expires_at: new Date(Date.now() + 15 * 60_000).toISOString(),
      account: { id: "a", email: "dana@example.org", username: "dreyes" },
    };
    atPTA(
      http.post("/api/v1/auth/login", () => HttpResponse.json(session)),
      http.post("/api/v1/auth/logout", () => HttpResponse.json({})),
    );
    renderAt("/login");
    await userEvent.type(await screen.findByLabelText("Username"), "dreyes");
    await userEvent.type(screen.getByLabelText("Password"), "velvet tractor oyster pilgrim");
    await userEvent.click(screen.getByRole("button", { name: "Log in" }));
    expect(await screen.findByText("dreyes")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Log out" }));
    expect(await screen.findByRole("heading", { name: "You are logged out" })).toBeInTheDocument();
  });

  it("a browser logged in to another organization is told so and offered a log out", async () => {
    server.use(
      http.get("/api/v1/address", () =>
        HttpResponse.json({
          ...PTA,
          logged_in_elsewhere: {
            name: "Riverside Community Committee",
            logout_url: "http://riverside.phoenixevent.localhost:5174/logout?return=x",
          },
        }),
      ),
    );
    renderAt("/login");
    expect(await screen.findByRole("alert")).toHaveTextContent("You are logged in to Riverside Community Committee.");
    expect(screen.getByRole("link", { name: "Log out" })).toHaveAttribute(
      "href",
      "http://riverside.phoenixevent.localhost:5174/logout?return=x",
    );
    expect(screen.queryByLabelText("Password")).not.toBeInTheDocument();
  });

  it("A3 shows each refused field under that field, then A4", async () => {
    let tries = 0;
    atPTA(
      http.post("/api/v1/auth/register", () => {
        tries++;
        if (tries === 1) {
          return HttpResponse.json(
            {
              code: "invalid_fields",
              message: "Check the fields marked below.",
              fields: [{ field: "password", message: "That password is too common. Choose another." }],
            },
            { status: 422 },
          );
        }
        return HttpResponse.json({ email: "lee@example.org" }, { status: 201 });
      }),
    );
    renderAt("/register");
    await userEvent.type(await screen.findByLabelText("Email address"), "lee@example.org");
    await userEvent.type(screen.getByLabelText("Password"), "password1234");
    await userEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("That password is too common. Choose another.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText(/We sent a confirmation link to lee@example.org/)).toBeInTheDocument();
  });

  it("E2: an ended reset link says so before a password is typed", async () => {
    atPTA(
      http.get("/api/v1/auth/link", () =>
        HttpResponse.json({ code: "link_ended", message: "This link has ended." }, { status: 410 }),
      ),
    );
    renderAt("/reset-password?token=abc");
    expect(await screen.findByRole("heading", { name: "This link has ended" })).toBeInTheDocument();
  });

  it("A6 names the account whose password is set", async () => {
    atPTA(http.get("/api/v1/auth/link", () => HttpResponse.json({ purpose: "set_password", login: "admin@example.org" })));
    renderAt("/set-password?token=abc");
    expect(await screen.findByText("For admin@example.org")).toBeInTheDocument();
  });

  it("a confirmation link confirms and shows the login page", async () => {
    atPTA(http.post("/api/v1/auth/confirm-email", () => HttpResponse.json({})));
    renderAt("/confirm-email?token=abc");
    expect(await screen.findByText("Your email address is confirmed. You can log in.")).toBeInTheDocument();
  });
});

describe("safeReturn", () => {
  const here = "riverside.phoenixevent.localhost";
  it("follows an address under the same domain", () => {
    expect(safeReturn("http://lincolnpta.phoenixevent.localhost:5174/login", here)).toBe(
      "http://lincolnpta.phoenixevent.localhost:5174/login",
    );
  });
  it("refuses any other address", () => {
    expect(safeReturn("https://evil.example/login", here)).toBeNull();
    expect(safeReturn("http://phoenixevent.localhost.evil.example/", here)).toBeNull();
    expect(safeReturn("javascript:alert(1)", here)).toBeNull();
    expect(safeReturn(null, here)).toBeNull();
  });
});
