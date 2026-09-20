import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";

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
      <Link className="ds-btn ds-btn--secondary" to="/dsa/dashboard">
        <Icon name="today" className="xl-ico--sm" /> Back to Today
      </Link>
    </>
  );
}
