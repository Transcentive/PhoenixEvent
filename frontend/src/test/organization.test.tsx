import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { DANA, SESSION, signedInAt } from "./app";

const profile = http.get("/api/v1/profile", () => HttpResponse.json(DANA));
const administrator = http.get("/api/v1/me", () =>
  HttpResponse.json({ account: SESSION.account, organization: {}, roles: ["organization_administrator"] }),
);
const OFFICERS = [
  { id: "a", name: "dreyes", roles: ["Organization Administrator"] },
  { id: "b", name: "Lee Park", roles: ["Organization Leadership"] },
];
const officers = http.get("/api/v1/officers", () => HttpResponse.json({ officers: OFFICERS }));

describe("O1, organization home", () => {
  it("links to the built organization pages and has no events yet", async () => {
    signedInAt("/", profile);
    expect(await screen.findByRole("heading", { level: 1, name: "Lincoln Elementary PTA" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Members and roles" })).toHaveAttribute("href", "/members");
    expect(screen.getByRole("link", { name: "Organization officers" })).toHaveAttribute("href", "/officers");
    expect(screen.getByText("No events yet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create event" })).not.toBeInTheDocument();
  });
});

describe("O2, Members and roles", () => {
  it("lists nobody until events exist", async () => {
    signedInAt("/members", profile);
    expect(await screen.findByText("No members listed yet")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to Lincoln Elementary PTA" })).toHaveAttribute("href", "/");
  });
});

describe("O8, Organization officers", () => {
  it("shows a member the officers without Change roles", async () => {
    signedInAt("/officers", profile, officers);
    expect(await screen.findByRole("link", { name: "Lee Park" })).toHaveAttribute("href", "/people/b");
    // Dana's own entry opens her own profile.
    const list = screen.getByText("Organization Leadership").closest("section")!;
    expect(within(list).getByRole("link", { name: "dreyes" })).toHaveAttribute("href", "/profile");
    expect(screen.getByText("Organization Leadership")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Change roles" })).not.toBeInTheDocument();
  });

  it("lets an Organization Administrator change roles in D5", async () => {
    let sent: unknown = null;
    signedInAt(
      "/officers",
      profile,
      officers,
      administrator,
      http.get("/api/v1/people/b/roles", () =>
        HttpResponse.json({ id: "b", name: "Lee Park", roles: ["organization_leadership"] }),
      ),
      http.put("/api/v1/people/b/roles", async ({ request }) => {
        sent = await request.json();
        return HttpResponse.json({ id: "b", name: "Lee Park", roles: ["organization_administrator", "organization_leadership"] });
      }),
    );
    const buttons = await screen.findAllByRole("button", { name: "Change roles" });
    await userEvent.click(buttons[1]!);
    const dialog = await screen.findByRole("dialog", { name: "Change roles for Lee Park" });
    expect(within(dialog).getByRole("checkbox", { name: "Member" })).toBeDisabled();
    expect(within(dialog).getByRole("checkbox", { name: "Organization Leadership" })).toBeChecked();
    await userEvent.click(within(dialog).getByRole("checkbox", { name: "Organization Administrator" }));
    await userEvent.click(within(dialog).getByRole("button", { name: "Save roles" }));
    expect(await screen.findByRole("link", { name: "Lee Park" })).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(sent).toEqual({ roles: ["organization_leadership", "organization_administrator"] });
  });

  it("shows the refusal to remove the last Organization Administrator", async () => {
    signedInAt(
      "/officers",
      profile,
      officers,
      administrator,
      http.get("/api/v1/people/a/roles", () =>
        HttpResponse.json({ id: "a", name: "dreyes", roles: ["organization_administrator"] }),
      ),
      http.put("/api/v1/people/a/roles", () =>
        HttpResponse.json(
          {
            code: "invalid_fields",
            message: "Check the fields marked below.",
            fields: [{ field: "roles", message: "dreyes is the last Organization Administrator of Lincoln Elementary PTA. Give another member Organization Administrator first." }],
          },
          { status: 422 },
        ),
      ),
    );
    await userEvent.click((await screen.findAllByRole("button", { name: "Change roles" }))[0]!);
    const dialog = await screen.findByRole("dialog", { name: "Change roles for dreyes" });
    await userEvent.click(within(dialog).getByRole("checkbox", { name: "Organization Administrator" }));
    await userEvent.click(within(dialog).getByRole("button", { name: "Save roles" }));
    expect(await within(dialog).findByText(/is the last Organization Administrator/)).toBeInTheDocument();
  });
});

describe("A9 and E3", () => {
  it("shows Change roles on a profile to an Organization Administrator", async () => {
    signedInAt("/people/b", profile, administrator, http.get("/api/v1/people/b", () => HttpResponse.json({ id: "b", name: "Lee Park" })));
    expect(await screen.findByRole("heading", { name: "Lee Park" })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Change roles" })).toBeInTheDocument();
  });

  it("answers a missing person and an unknown page with E3", async () => {
    signedInAt("/people/zzz", profile, http.get("/api/v1/people/zzz", () => HttpResponse.json({ code: "not_found" }, { status: 404 })));
    expect(await screen.findByRole("heading", { name: "This page is outside your access" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go to organization home" })).toHaveAttribute("href", "/");
  });

  it("answers an unknown address with E3", async () => {
    signedInAt("/no-such-page", profile);
    expect(await screen.findByRole("heading", { name: "This page is outside your access" })).toBeInTheDocument();
  });
});
