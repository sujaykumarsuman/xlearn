import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { routes } from "../router";

function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path], basename: "/xlearn" });
  const queryClient = new QueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe("AppShell + routing", () => {
  it("renders the shell chrome and the routed screen", async () => {
    renderAt("/xlearn/dsa/dashboard");
    // Sidebar brand + nav (chrome)
    expect(screen.getByRole("navigation")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /roadmap/i })).toBeInTheDocument();
    // Routed screen body
    expect(await screen.findByRole("heading", { level: 1, name: "Today" })).toBeInTheDocument();
  });

  it("marks the active nav item for the current route", () => {
    renderAt("/xlearn/dsa/revision");
    const active = document.querySelector(".xl-nav__item--active");
    expect(active?.textContent).toContain("Revision");
  });

  it("renders params-driven screens", () => {
    renderAt("/xlearn/dsa/problem/16");
    expect(screen.getByRole("heading", { level: 1, name: "Problem #16" })).toBeInTheDocument();
  });

  it("expanded: the path switcher opens the dropdown", () => {
    renderAt("/xlearn/dsa/dashboard");
    fireEvent.click(screen.getByRole("button", { name: /DSA Interview Mastery/i }));
    expect(screen.getByText("Browse all paths")).toBeInTheDocument();
  });

  it("collapsed: the path switcher navigates to Catalog instead of the cramped menu", async () => {
    renderAt("/xlearn/dsa/dashboard");
    // Emulate the pure-CSS collapsed state (the checkbox is the source of truth).
    const checkbox = document.getElementById("xl-collapse") as HTMLInputElement;
    checkbox.checked = true;
    fireEvent.click(screen.getByRole("button", { name: /DSA Interview Mastery/i }));
    expect(await screen.findByRole("heading", { level: 1, name: "Learning paths" })).toBeInTheDocument();
    expect(screen.queryByText("Browse all paths")).not.toBeInTheDocument();
  });

  it("renders all 12 routes (plus the 404) inside the shell without crashing", () => {
    const paths = [
      "/xlearn",
      "/xlearn/dsa",
      "/xlearn/dsa/dashboard",
      "/xlearn/dsa/week/2",
      "/xlearn/dsa/concept/sliding-window",
      "/xlearn/dsa/problem/16",
      "/xlearn/dsa/revision",
      "/xlearn/dsa/mistakes",
      "/xlearn/dsa/mock",
      "/xlearn/dsa/progress",
      "/xlearn/settings",
      "/xlearn/auth",
      "/xlearn/nope-404",
    ];
    for (const path of paths) {
      const { unmount } = renderAt(path);
      expect(document.querySelector(".xl-app")).not.toBeNull();
      unmount();
    }
  });
});
