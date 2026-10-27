import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "./server";
import { DANA, renderAt } from "./app";

const SESSION = {
  access_token: "t",
  expires_at: new Date(Date.now() + 15 * 60_000).toISOString(),
  account: { id: "a", email: "dana@example.org", username: "dreyes" },
};

function signedInAt(path: string, ...handlers: Parameters<typeof server.use>) {
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
  );
  renderAt(path);
}

describe("A8, the profile and its visibility", () => {
  it("saves a personal name, and the header shows it", async () => {
    let sent: Record<string, string> = {};
    signedInAt(
      "/profile",
      http.get("/api/v1/profile", () => HttpResponse.json(DANA)),
      http.put("/api/v1/profile", async ({ request }) => {
        sent = (await request.json()) as Record<string, string>;
        return HttpResponse.json({
          ...DANA,
          first_name: "Dana",
          last_name: "Reyes",
          name_shown: "personal_name",
          name: "Dana Reyes",
        });
      }),
    );
    expect(await screen.findByRole("link", { name: "dreyes" })).toHaveAttribute("href", "/profile");
    await userEvent.type(await screen.findByLabelText("First name"), "Dana");
    await userEvent.type(screen.getByLabelText("Last name"), "Reyes");
    await userEvent.click(screen.getByRole("button", { name: "Save profile" }));
    expect(await screen.findByText("Your profile is saved.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Dana Reyes" })).toBeInTheDocument();
    // The profile section sends the profile only.
    expect(sent).toEqual({ username: "dreyes", first_name: "Dana", middle_name: "", last_name: "Reyes" });
    // The saved personal name can now be chosen as the name shown.
    expect(screen.getByRole("radio", { name: "Personal name: Dana Reyes" })).toBeEnabled();
  });

  it("warns beneath a level set to public, and shows a refusal under its field", async () => {
    signedInAt(
      "/profile",
      http.get("/api/v1/profile", () => HttpResponse.json(DANA)),
      http.put("/api/v1/profile/visibility", () =>
        HttpResponse.json(
          {
            code: "invalid_fields",
            message: "Check the fields marked below.",
            fields: [{ field: "name_shown", message: "The name shown must be visible at least to organization members." }],
          },
          { status: 422 },
        ),
      ),
    );
    await userEvent.selectOptions(await screen.findByRole("combobox", { name: "Username" }), "public");
    expect(screen.getByRole("note")).toHaveTextContent("including people outside Lincoln Elementary PTA");
    // A name not saved in the profile cannot be chosen.
    expect(screen.getByRole("radio", { name: "Personal name: not saved" })).toBeDisabled();
    await userEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText(/must be visible at least to organization members\.$/)).toBeInTheDocument();
  });

  it("changes the email address with the current password", async () => {
    signedInAt(
      "/profile",
      http.get("/api/v1/profile", () => HttpResponse.json(DANA)),
      http.post("/api/v1/profile/email", () => HttpResponse.json({ ...DANA, email: "dana@new.example" })),
    );
    await userEvent.click(await screen.findByRole("button", { name: "Change" }));
    await userEvent.type(screen.getByLabelText("New email address"), "dana@new.example");
    await userEvent.type(screen.getByLabelText("Current password"), "velvet tractor oyster pilgrim");
    await userEvent.click(screen.getByRole("button", { name: "Change email address" }));
    expect(await screen.findByText(/Your email address is changed/)).toBeInTheDocument();
    expect(screen.getByText("dana@new.example")).toBeInTheDocument();
  });
});

describe("A9, another person's profile", () => {
  it("shows only what is shared, with no Invite member yet", async () => {
    signedInAt(
      "/people/b",
      http.get("/api/v1/profile", () => HttpResponse.json(DANA)),
      http.get("/api/v1/people/b", () =>
        HttpResponse.json({ id: "b", name: "Lee Park", personal_name: "Lee Park", roles: ["Organization Leadership"] }),
      ),
    );
    expect(await screen.findByRole("heading", { name: "Lee Park" })).toBeInTheDocument();
    expect(screen.getByText("Organization Leadership")).toBeInTheDocument();
    expect(screen.queryByText("Email address")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Invite member" })).not.toBeInTheDocument();
  });
});
