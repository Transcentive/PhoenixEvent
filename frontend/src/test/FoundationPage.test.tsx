import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import FoundationPage from "@/pages/FoundationPage";
import { server } from "./server";

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <FoundationPage />
    </QueryClientProvider>,
  );
}

describe("FoundationPage", () => {
  it("says Ready when the API and the database answer", async () => {
    server.use(
      http.get("/api/v1/health", () =>
        HttpResponse.json({ status: "ok", database: "ok", build: "abc123" }),
      ),
    );
    renderPage();

    expect(screen.getByText("Checking")).toBeInTheDocument();
    expect(await screen.findByText("Ready")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Build abc123");
  });

  it("says Unavailable when the database does not answer", async () => {
    server.use(
      http.get("/api/v1/health", () =>
        HttpResponse.json(
          { status: "unavailable", database: "unreachable", build: "abc123" },
          { status: 503 },
        ),
      ),
    );
    renderPage();

    expect(await screen.findByText("Unavailable")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Start PostgreSQL");
  });

  it("says Unavailable when the API does not answer", async () => {
    server.use(http.get("/api/v1/health", () => HttpResponse.error()));
    renderPage();

    expect(await screen.findByText("Unavailable")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Start it");
  });
});
