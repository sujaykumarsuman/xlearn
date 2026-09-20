import { Placeholder } from "../components/Placeholder";

export default function Auth() {
  return (
    <Placeholder
      eyebrow="Welcome"
      title="Sign in"
      icon="key"
      sprint="S02"
      summary="OAuth with GitHub or Google, then a 3-step onboarding (path → study budget → API key)."
    >
      <p className="xl-mut" style={{ fontSize: 12, marginTop: 8 }}>
        In S02 this moves to a standalone pre-auth layout (no app shell); it renders inside the shell
        here only so the scaffold shows all 12 routes.
      </p>
    </Placeholder>
  );
}
