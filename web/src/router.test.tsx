import { isValidElement } from "react";
import { matchRoutes } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { CurriculumShell, PlainShell } from "./components/AppShell";
import { isCourseNotFoundHandle } from "./lib/course";
import { routes } from "./router";
import Auth from "./screens/Auth";
import Catalog from "./screens/Catalog";
import Concept from "./screens/Concept";
import Dashboard from "./screens/Dashboard";
import NotFound from "./screens/NotFound";
import Problem from "./screens/Problem";
import Problems from "./screens/Problems";
import Roadmap from "./screens/Roadmap";
import Settings from "./screens/Settings";
import UserDashboard from "./screens/UserDashboard";
import Week from "./screens/Week";

/** resolve matches an app-relative path and returns the leaf screen, the layout shell it
 *  renders in, the params, and whether it is the course subtree's catch-all. */
function resolve(path: string) {
  const matches = matchRoutes(routes, path) ?? [];
  const types = matches.map((m) => (isValidElement(m.route.element) ? m.route.element.type : undefined));
  const leaf = matches[matches.length - 1];
  return {
    screen: types[types.length - 1],
    shell: types.includes(CurriculumShell) ? "curriculum" : types.includes(PlainShell) ? "plain" : "none",
    params: leaf?.params ?? {},
    courseCatchAll: matches.some((m) => isCourseNotFoundHandle(m.route.handle)),
  };
}

describe("router ranking: the dynamic :course segment never shadows a static route", () => {
  it.each([
    ["/", Catalog, "plain"],
    ["/settings", Settings, "plain"],
    ["/auth", Auth, "none"],
    ["/u/ada", UserDashboard, "none"],
  ])("%s → its static screen", (path, screen, shell) => {
    const r = resolve(path);
    expect(r.screen).toBe(screen);
    expect(r.shell).toBe(shell);
    expect(r.params.course).toBeUndefined();
  });

  it.each([
    ["/dsa", Roadmap, {}],
    ["/dsa/dashboard", Dashboard, {}],
    ["/dsa/problems", Problems, {}],
    ["/dsa/week/3", Week, { n: "3" }],
    ["/dsa/concept/two-pointers", Concept, { slug: "two-pointers" }],
    ["/dsa/problem/16", Problem, { id: "16" }],
    ["/zz-fixture/week/2", Week, { n: "2" }],
  ])("%s → the course subtree", (path, screen, params) => {
    const r = resolve(path);
    expect(r.screen).toBe(screen);
    expect(r.shell).toBe("curriculum");
    expect(r.params).toMatchObject({ course: path.split("/")[1], ...params });
    expect(r.courseCatchAll).toBe(false);
  });

  it("sends an unknown single segment to the course subtree (resolved to NotFound by the catalog)", () => {
    const r = resolve("/nope");
    expect(r.screen).toBe(Roadmap);
    expect(r.shell).toBe("curriculum");
    expect(r.params.course).toBe("nope");
  });

  it("sends an unknown sub-route of a course to the subtree's catch-all (NotFound in the plain frame)", () => {
    for (const path of ["/dsa/bogus", "/dsa/week", "/dsa/dashboard/extra", "/nope/404", "/settings/x"]) {
      const r = resolve(path);
      expect(r.screen, path).toBe(NotFound);
      expect(r.courseCatchAll, path).toBe(true);
    }
  });
});
