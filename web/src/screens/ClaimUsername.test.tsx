import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { authedMe, installFetchMock, restoreFetch } from "../test/fetchMock";

function renderApp(initialPath: string) {
  const router = createMemoryRouter(routes, { initialEntries: [initialPath], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

/** An authed /me with an optional username on the account. */
function meWithUsername(username?: string) {
  const me = authedMe("dsa");
  return { ...me, account: { ...me.account, username } };
}

describe("ClaimUsername (/xlearn/claim-username)", () => {
  afterEach(restoreFetch);

  it("redirects to the public dashboard when a username already exists", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: meWithUsername("ada") };
      if (url.includes("/api/u/ada"))
        return {
          status: 200,
          body: {
            user: { username: "ada", displayName: "Ada Lovelace", joinedAt: "2026-01-01T00:00:00Z" },
            totals: { solved: 0, streak: { current: 0, longest: 0 } },
            mock: { count: 0, best: 0, average: 0 },
            heatmap: null,
            courses: [],
          },
        };
      return { status: 404 };
    });
    renderApp("/xlearn/claim-username");
    // Redirected straight to the (public) profile.
    expect(await screen.findByText("@ada")).toBeInTheDocument();
  });

  it("claims a username: checks availability, posts, then lands on the profile", async () => {
    let posted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: meWithUsername(undefined) };
      if (url.includes("/api/username/available")) return { status: 200, body: { available: true } };
      if (url.includes("/api/me/username")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { ok: true, username: "newbie" } };
      }
      if (url.includes("/api/u/newbie"))
        return {
          status: 200,
          body: {
            user: { username: "newbie", displayName: "Ada Lovelace", joinedAt: "2026-01-01T00:00:00Z" },
            totals: { solved: 0, streak: { current: 0, longest: 0 } },
            mock: { count: 0, best: 0, average: 0 },
            heatmap: null,
            courses: [],
          },
        };
      return { status: 404 };
    });
    renderApp("/xlearn/claim-username");

    await screen.findByRole("heading", { name: /claim your username/i });
    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "newbie" } });

    // Debounced availability probe resolves to available.
    expect(await screen.findByText(/@newbie is available/i)).toBeInTheDocument();

    const claim = screen.getByRole("button", { name: /claim username/i });
    await waitFor(() => expect(claim).toBeEnabled());
    fireEvent.click(claim);

    // Posted the normalised username, then navigated to the fresh public profile.
    await waitFor(() => expect(posted).toEqual({ username: "newbie" }));
    expect(await screen.findByText("@newbie")).toBeInTheDocument();
  });
});
