import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { coursePath } from "../lib/course";
import { DEFAULT_COURSE } from "../lib/defaultCourse";

/** NotFound is v1's 404, unchanged (AB02-F6): an unknown route, an unknown / retired /
 *  hidden course, or an unknown sub-route of a course all render it, in the plain frame,
 *  with no hint whether a slug exists. "Back to Today" keeps v1's target, the default
 *  course's dashboard (the SPA's one default, lib/defaultCourse.ts). */
export default function NotFound() {
  return (
    <>
      <div className="xl-page-h">
        <div>
          <div className="xl-eyebrow">404</div>
          <h1 style={{ marginTop: 6 }}>Page not found</h1>
          <p>That route doesn’t exist in xLearn.</p>
        </div>
      </div>
      <Link className="ds-btn ds-btn--secondary" to={coursePath(DEFAULT_COURSE, "dashboard")}>
        <Icon name="today" className="xl-ico--sm" /> Back to Today
      </Link>
    </>
  );
}
