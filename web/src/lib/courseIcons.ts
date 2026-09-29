// Per-course display icon: presentation only. The catalog (GET /paths) decides which
// courses exist and their state; this map only picks a glyph for the ones we know, and
// every other course gets the neutral grid. Used by the Catalog cards, the onboarding path
// picker and the coming-soon teaser (AB02-F4), so one course shows one icon everywhere.
import type { IconName } from "../components/Icon";

const COURSE_ICON: Record<string, IconName> = {
  dsa: "code",
  "system-design": "server",
  "go-concurrency": "branch",
  "lld-ood": "layers",
  sql: "db",
  behavioral: "chat",
};

/** courseIcon returns a course's display icon (the grid for an unknown course). */
export function courseIcon(slug: string): IconName {
  return Object.hasOwn(COURSE_ICON, slug) ? COURSE_ICON[slug]! : "grid";
}
