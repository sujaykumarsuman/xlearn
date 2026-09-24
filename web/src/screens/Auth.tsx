import { Fragment, useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Icon, IconSprite } from "../components/Icon";
import { Spinner } from "../components/States";
import { BudgetFields } from "../components/BudgetFields";
import { DEFAULT_WEEKDAY, DEFAULT_WEEKEND, budgetEta, clampWeekday } from "../lib/budget";
import { type ApiRequestError } from "../lib/api";
import {
  oauthStartAction,
  useCompleteOnboarding,
  useDevAuthEnabled,
  useDevLogin,
  useLogin,
  useMe,
  useSetOnboardingBudget,
  useSetOnboardingPath,
  useSetUsername,
  useSignup,
  useUsernameAvailability,
  type Me,
  type StudyBudget,
  type WeekendBand,
} from "../lib/auth";
import { COACH_MODELS, COACH_PROVIDERS, coachModelLabel, usePutCoachKey, type ProviderId } from "../lib/settings";

/**
 * Auth is the standalone pre-auth screen (no app shell): a two-column layout with a
 * marketing panel and either sign-in (GitHub OAuth + email/username + password, ADR-0023)
 * or the 4-step onboarding. Onboarding: path (step 1) · study budget (step 2, persisted) ·
 * username (step 3, optional — claim your public @handle, F009) · coach key (step 4,
 * optional — Skip/Finish completes onboarding). The learner resumes at the first unfinished
 * step and is routed into the app once complete.
 */
export default function Auth() {
  const me = useMe();

  // Data-first: whenever /me has ever resolved we keep the user in onboarding, even
  // if a later background refetch (e.g. the invalidate after saving step 1) briefly
  // fails — a transient error must not eject an authenticated user to sign-in.
  let right: React.ReactNode;
  if (me.data) {
    right = <Onboarding me={me.data} />;
  } else if (me.isLoading) {
    right = <Centered>Loading…</Centered>;
  } else {
    // Genuinely unauthenticated (401) or errored with no cached session.
    right = <SignIn />;
  }

  return (
    <div className="xl-auth">
      <IconSprite />
      <LeftPanel />
      <div className="xl-auth__right">
        <div className="xl-auth__card">{right}</div>
      </div>
    </div>
  );
}

// --- Left marketing panel (inline-styled per the artboard) ---

function LeftPanel() {
  return (
    <div className="xl-auth__left">
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <span className="xl-brand__mark" style={{ width: 36, height: 36, fontSize: 19 }}>
          x
        </span>
        <span style={{ fontSize: 20, fontWeight: 700 }}>
          x<b style={{ color: "var(--ds-teal)" }}>Learn</b>
        </span>
      </div>

      <div style={{ margin: "auto 0", maxWidth: 440 }}>
        <h1 style={{ fontSize: 34, fontWeight: 700, lineHeight: 1.15 }}>
          Master the method,
          <br />
          not just the problems.
        </h1>
        <p style={{ fontSize: 15, color: "var(--ds-dim)", marginTop: 12 }}>
          A guided, self-paced course that enforces how you learn — timed attempts, gated hints, and
          spaced revision that actually sticks.
        </p>

        <div style={{ display: "grid", gap: 14, marginTop: 26 }}>
          <Feature icon="code" title="Guided problem stages" desc="Attempt → hint → solution, gated by a timer." />
          <Feature icon="refresh" title="Five-touch spaced revision" desc="Re-solve on Day 1 · 3 · 7 · 21 · 45 — or reset." />
          <Feature icon="target" title="Scored mock interviews" desc="45-min rounds on a 7-dimension rubric." />
        </div>

        <div
          style={{
            marginTop: 26,
            background: "rgba(18,21,28,.7)",
            border: "1px solid var(--ds-line)",
            borderRadius: 12,
            padding: "14px 16px",
            display: "flex",
            alignItems: "center",
            gap: 12,
          }}
        >
          <div className="xl-touch" style={{ flex: "none" }}>
            <span className="xl-touch__d xl-touch__d--pass" />
            <span className="xl-touch__d xl-touch__d--pass" />
            <span className="xl-touch__d xl-touch__d--due" />
            <span className="xl-touch__d xl-touch__d--mock" />
            <span className="xl-touch__d xl-touch__d--mock" />
          </div>
          <span style={{ fontSize: 12, color: "var(--ds-dim)" }}>
            The signature retention loop — <b style={{ color: "var(--ds-text)" }}>151 problems</b>, remembered.
          </span>
        </div>
      </div>

      <div style={{ fontFamily: "var(--ds-font-mono)", fontSize: 11.5, color: "var(--ds-muted)" }}>
        projects.sujaykumar.dev/xlearn
      </div>
    </div>
  );
}

function Feature({ icon, title, desc }: { icon: "code" | "refresh" | "target"; title: string; desc: string }) {
  return (
    <div style={{ display: "flex", gap: 12, alignItems: "flex-start" }}>
      <span
        style={{
          flex: "none",
          width: 34,
          height: 34,
          borderRadius: 9,
          display: "grid",
          placeItems: "center",
          background: "rgba(53,208,192,.14)",
          color: "var(--ds-teal)",
        }}
      >
        <Icon name={icon} className="xl-ico--sm" />
      </span>
      <div>
        <b style={{ fontSize: 13.5 }}>{title}</b>
        <div style={{ fontSize: 12.5, color: "var(--ds-muted)" }}>{desc}</div>
      </div>
    </div>
  );
}

// --- Sign-in view (OAuth-only) ---

function SignIn() {
  const [params] = useSearchParams();
  const error = params.get("error");
  const devEnabled = useDevAuthEnabled();
  const devLogin = useDevLogin();

  const [mode, setMode] = useState<"signin" | "signup">("signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useLogin();
  const signup = useSignup();
  const pending = login.isPending || signup.isPending;
  const err = mode === "signin" ? login.error : signup.error;

  // Login accepts an email OR a username (F009); signup still requires a real email. So the
  // identifier is validated strictly only in signup mode.
  const identifier = email.trim();
  const emailOk = /^\S+@\S+\.\S+$/.test(identifier);
  const identifierOk = mode === "signup" ? emailOk : identifier.length > 0;
  const canSubmit = identifierOk && password.length >= (mode === "signup" ? 8 : 1) && !pending;

  const clearErrors = () => {
    login.reset();
    signup.reset();
  };
  const switchMode = (m: "signin" | "signup") => {
    setMode(m);
    clearErrors();
  };
  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    (mode === "signup" ? signup : login).mutate({ email: identifier, password });
  };

  return (
    <>
      <h2 style={{ fontSize: 24, fontWeight: 700 }}>Sign in to xLearn</h2>
      <p style={{ fontSize: 13, color: "var(--ds-muted)", marginTop: 4 }}>
        Continue with GitHub, or use your email.
      </p>

      {error && (
        <div
          role="alert"
          style={{
            marginTop: 14,
            padding: "10px 12px",
            borderRadius: 9,
            border: "1px solid var(--ds-err)",
            background: "rgba(242,109,109,.08)",
            color: "var(--ds-err)",
            fontSize: 12.5,
          }}
        >
          {oauthErrorMessage(error)}
        </div>
      )}

      <div style={{ display: "grid", gap: 10, marginTop: 18 }}>
        <form method="post" action={oauthStartAction("github")}>
          <button type="submit" className="ds-btn ds-btn--secondary ds-btn--block">
            <GitHubMark /> Continue with GitHub
          </button>
        </form>

        {/* Local-only dev sign-in (F002 / ADR-0022): rendered only when the gateway
            reports DEV_AUTH is on — never in a prod image. */}
        {devEnabled.data && (
          <button
            type="button"
            className="ds-btn ds-btn--ghost ds-btn--block"
            disabled={devLogin.isPending}
            onClick={() => devLogin.mutate()}
          >
            <Icon name="key" className="xl-ico--sm" />{" "}
            {devLogin.isPending ? "Signing in…" : "Dev sign in (local)"}
          </button>
        )}
        {devLogin.isError && (
          <p role="alert" style={{ fontSize: 12, color: "var(--ds-err)", margin: 0 }}>
            Dev sign-in failed. Is DEV_AUTH enabled?
          </p>
        )}
      </div>

      <div style={{ display: "flex", alignItems: "center", gap: 10, margin: "18px 0" }}>
        <span style={{ flex: 1, height: 1, background: "var(--ds-line)" }} />
        <span style={{ fontSize: 11, color: "var(--ds-muted)" }}>or</span>
        <span style={{ flex: 1, height: 1, background: "var(--ds-line)" }} />
      </div>

      {/* Sign in / Sign up selector (ADR-0023): full-width, circular-ended pill. */}
      <div className="ds-seg ds-seg--block ds-seg--pill" role="group" aria-label="Sign in or sign up" style={{ marginBottom: 14 }}>
        <button type="button" className={mode === "signin" ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"} aria-pressed={mode === "signin"} onClick={() => switchMode("signin")}>
          Sign in
        </button>
        <button type="button" className={mode === "signup" ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"} aria-pressed={mode === "signup"} onClick={() => switchMode("signup")}>
          Sign up
        </button>
      </div>

      <form onSubmit={submit}>
        <div className="ds-field">
          <label className="ds-field__label" htmlFor="email">
            {mode === "signup" ? "Email" : "Email or username"}
          </label>
          <input
            id="email"
            className="ds-input"
            type={mode === "signup" ? "email" : "text"}
            placeholder={mode === "signup" ? "you@example.com" : "you@example.com or your-handle"}
            autoComplete={mode === "signup" ? "email" : "username"}
            autoCapitalize="none"
            spellCheck={false}
            value={email}
            onChange={(e) => {
              clearErrors();
              setEmail(e.target.value);
            }}
          />
        </div>
        <div className="ds-field" style={{ marginTop: 12 }}>
          <label className="ds-field__label" htmlFor="password">
            Password
          </label>
          <input
            id="password"
            className="ds-input"
            type="password"
            placeholder="••••••••••"
            autoComplete={mode === "signup" ? "new-password" : "current-password"}
            value={password}
            onChange={(e) => {
              clearErrors();
              setPassword(e.target.value);
            }}
          />
          {mode === "signup" && (
            <p style={{ fontSize: 11, color: "var(--ds-muted)", margin: "6px 0 0" }}>At least 8 characters.</p>
          )}
        </div>
        {err && (
          <p role="alert" style={{ fontSize: 12.5, color: "var(--ds-err)", margin: "10px 0 0" }}>
            {emailAuthErrorMessage(err, mode)}
          </p>
        )}
        <button type="submit" className="ds-btn ds-btn--primary ds-btn--block ds-btn--lg" style={{ marginTop: 14 }} disabled={!canSubmit}>
          {pending ? (mode === "signup" ? "Creating account…" : "Signing in…") : mode === "signup" ? "Create account" : "Log in"}
        </button>
      </form>
    </>
  );
}

function oauthErrorMessage(code: string): string {
  switch (code) {
    case "denied":
      return "Sign-in was cancelled. Try again to continue.";
    case "oauth_state":
    case "oauth_code":
      return "That sign-in link expired. Please try again.";
    case "link_auth":
      return "Please sign in first, then connect GitHub from Settings.";
    case "account_exists_password":
      return "An account with this email already uses a password. Sign in with your password, then connect GitHub from Settings.";
    case "signup_closed":
      return "xLearn is invite-only right now. This GitHub account isn’t connected to an xLearn account.";
    default:
      return "Something went wrong signing in. Please try again.";
  }
}

function emailAuthErrorMessage(err: ApiRequestError, mode: "signin" | "signup"): string {
  switch (err.code) {
    case "email_taken":
      return "That email is already registered — switch to Sign in.";
    case "signup_closed":
      return "xLearn is invite-only right now.";
    case "invalid_credentials":
      return "Incorrect email/username or password.";
    case "weak_password":
      return "Password must be 8–72 characters.";
    case "invalid_email":
      return "Enter a valid email address.";
    default:
      return mode === "signup" ? "Couldn’t create your account. Please try again." : "Couldn’t sign you in. Please try again.";
  }
}

// --- Onboarding (4-step; path + budget + username persist; coach step 4 is a preview) ---

function Onboarding({ me }: { me: Me }) {
  const navigate = useNavigate();
  // Returning, fully-onboarded users skip straight into the app; everyone else resumes
  // at the first unfinished step. Both are computed ONCE from the initial /me so a
  // background refetch (the invalidate after saving a step) can't reset the flow.
  const [alreadyDone] = useState(() => me.onboarding.completed);
  const [step, setStep] = useState(() => firstUnfinishedStep(me));
  const [path, setPath] = useState(me.onboarding.path_chosen ?? "dsa");
  const setOnboardingPath = useSetOnboardingPath();

  useEffect(() => {
    if (alreadyDone) navigate("/", { replace: true });
  }, [alreadyDone, navigate]);

  if (alreadyDone) return <Centered>Taking you in…</Centered>;

  const enterApp = () => navigate("/", { replace: true });

  return (
    <>
      <Stepper step={step} />
      {step === 1 && (
        <StepPath
          path={path}
          onPick={setPath}
          pending={setOnboardingPath.isPending}
          error={setOnboardingPath.isError}
          onContinue={() => setOnboardingPath.mutate(path, { onSuccess: () => setStep(2) })}
        />
      )}
      {step === 2 && <StepBudget initial={me.account.study_budget} onBack={() => setStep(1)} onContinue={() => setStep(3)} />}
      {step === 3 && <StepUsername currentUsername={me.account.username} onBack={() => setStep(2)} onContinue={() => setStep(4)} />}
      {step === 4 && <StepCoach onBack={() => setStep(3)} onFinish={enterApp} />}
    </>
  );
}

/** firstUnfinishedStep maps onboarding state to the step to resume at (1..3). */
function firstUnfinishedStep(me: Me): number {
  if (!me.onboarding.path_chosen) return 1;
  if (!me.onboarding.budget_set) return 2;
  return 3;
}

function Stepper({ step, total = 4 }: { step: number; total?: number }) {
  const dot = (n: number) =>
    step > n ? "xl-touch__d xl-touch__d--pass" : step === n ? "xl-touch__d xl-touch__d--due" : "xl-touch__d";
  const bar = (n: number) => (step > n ? "var(--ds-teal)" : "var(--ds-line-2)");
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 26 }}>
      {Array.from({ length: total }, (_, i) => i + 1).map((n) => (
        <Fragment key={n}>
          <span className={dot(n)} />
          {n < total && <span style={{ flex: 1, height: 2, background: bar(n) }} />}
        </Fragment>
      ))}
    </div>
  );
}

function StepPath({
  path,
  onPick,
  onContinue,
  pending,
  error,
}: {
  path: string;
  onPick: (p: string) => void;
  onContinue: () => void;
  pending: boolean;
  error: boolean;
}) {
  const dsaSelected = path === "dsa";
  return (
    <>
      <div className="xl-eyebrow">Step 1 of 4</div>
      <h2 style={{ fontSize: 22, fontWeight: 700, marginTop: 6 }}>Pick your path</h2>
      <p style={{ fontSize: 13, color: "var(--ds-muted)", marginTop: 4 }}>
        Start with DSA — more paths are on the way.
      </p>

      <button
        type="button"
        onClick={() => onPick("dsa")}
        aria-pressed={dsaSelected}
        className={dsaSelected ? "ds-card ds-card--teal" : "ds-card"}
        style={{ display: "flex", alignItems: "center", gap: 13, padding: 15, borderRadius: 11, marginTop: 16, width: "100%", textAlign: "left", cursor: "pointer" }}
      >
        <span style={{ flex: "none", width: 38, height: 38, borderRadius: 9, display: "grid", placeItems: "center", background: "rgba(53,208,192,.14)", color: "var(--ds-teal)" }}>
          <Icon name="code" />
        </span>
        <span style={{ flex: 1 }}>
          <b style={{ fontSize: 14 }}>Data Structures &amp; Algorithms</b>
          <br />
          <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>16 weeks · 151 problems · Go-first</span>
        </span>
        {dsaSelected && (
          <span style={{ flex: "none", width: 22, height: 22, borderRadius: 999, display: "grid", placeItems: "center", background: "var(--ds-teal)", color: "#06231f" }}>
            <Icon name="check" className="xl-ico--sm" />
          </span>
        )}
      </button>

      <div
        style={{ display: "flex", alignItems: "center", gap: 13, padding: 15, borderRadius: 11, marginTop: 10, border: "1px dashed var(--ds-line-2)", opacity: 0.6 }}
      >
        <span style={{ flex: "none", width: 38, height: 38, borderRadius: 9, display: "grid", placeItems: "center", background: "var(--ds-inset)", color: "var(--ds-muted)" }}>
          <Icon name="lock" />
        </span>
        <span style={{ flex: 1 }}>
          <b style={{ fontSize: 14 }}>System Design · Go Concurrency · +3</b>
          <br />
          <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>Coming soon</span>
        </span>
      </div>

      {error && (
        <p style={{ color: "var(--ds-err)", fontSize: 12.5, marginTop: 12 }}>
          Couldn’t save your path. Please try again.
        </p>
      )}

      <button
        type="button"
        className="ds-btn ds-btn--primary ds-btn--block ds-btn--lg"
        style={{ marginTop: 22 }}
        disabled={pending || !dsaSelected}
        onClick={onContinue}
      >
        {pending ? "Saving…" : "Continue"} <Icon name="arrow" className="xl-ico--sm" />
      </button>
    </>
  );
}

function StepBudget({ initial, onBack, onContinue }: { initial: StudyBudget; onBack: () => void; onContinue: () => void }) {
  const save = useSetOnboardingBudget();
  // Seed from any already-saved budget so returning to this step (Back from step 3)
  // doesn't reset the picker to the defaults and overwrite the saved value on Continue.
  const [weekday, setWeekday] = useState(clampWeekday(initial.weekday_minutes ?? DEFAULT_WEEKDAY));
  const [weekend, setWeekend] = useState<WeekendBand>(initial.weekend_band ?? DEFAULT_WEEKEND);

  return (
    <>
      <div className="xl-eyebrow">Step 2 of 4</div>
      <h2 style={{ fontSize: 22, fontWeight: 700, marginTop: 6 }}>Set your study budget</h2>
      <p style={{ fontSize: 13, color: "var(--ds-muted)", marginTop: 4 }}>
        We’ll size each day’s plan to fit. You can change this anytime.
      </p>

      <div style={{ marginTop: 22 }}>
        <BudgetFields weekday={weekday} weekend={weekend} onWeekday={setWeekday} onWeekend={setWeekend} />
        <div
          style={{
            marginTop: 18,
            padding: "12px 14px",
            background: "var(--ds-inset)",
            borderRadius: 9,
            fontSize: 12,
            color: "var(--ds-dim)",
            display: "flex",
            gap: 8,
            alignItems: "center",
          }}
        >
          <Icon name="clock" className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} /> ~{budgetEta(weekday, weekend)} to
          interview-ready at this pace.
        </div>
      </div>

      {save.isError && (
        <p style={{ color: "var(--ds-err)", fontSize: 12.5, marginTop: 12 }}>Couldn’t save your budget. Please try again.</p>
      )}

      <div style={{ display: "flex", gap: 10, marginTop: 22 }}>
        <button type="button" className="ds-btn ds-btn--ghost" onClick={onBack} disabled={save.isPending}>
          Back
        </button>
        <button
          type="button"
          className="ds-btn ds-btn--primary ds-btn--lg"
          style={{ flex: 1 }}
          disabled={save.isPending}
          onClick={() => save.mutate({ weekday_minutes: weekday, weekend_band: weekend }, { onSuccess: onContinue })}
        >
          {save.isPending ? "Saving…" : "Continue"} <Icon name="arrow" className="xl-ico--sm" />
        </button>
      </div>
    </>
  );
}

function StepUsername({
  currentUsername,
  onBack,
  onContinue,
}: {
  currentUsername?: string;
  onBack: () => void;
  onContinue: () => void;
}) {
  const setU = useSetUsername();
  const [value, setValue] = useState(currentUsername ?? "");
  const [debounced, setDebounced] = useState("");

  // Debounce the availability probe so it doesn't fire on every keystroke.
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value.trim().toLowerCase()), 350);
    return () => clearTimeout(t);
  }, [value]);
  const avail = useUsernameAvailability(debounced);

  const normalized = value.trim().toLowerCase();
  const changed = normalized !== (currentUsername ?? "");
  // `settled` guards the debounce window: only enable a claim once the availability probe
  // reflects the CURRENT input (avail is keyed on `debounced`, which trails `value` by 350ms).
  const settled = debounced === normalized;
  const canClaim = normalized.length >= 3 && changed && settled && avail.data?.available === true && !setU.isPending;

  // Optional step: claim + advance when a valid name is entered, else Skip. If the user
  // already has a username and didn't change it, the primary button just continues.
  const primary = () => {
    if (canClaim) {
      setU.mutate(normalized, { onSuccess: onContinue });
      return;
    }
    if (currentUsername && !changed) onContinue();
  };

  return (
    <>
      <div className="xl-eyebrow">Step 3 of 4 · optional</div>
      <h2 style={{ fontSize: 22, fontWeight: 700, marginTop: 6 }}>Claim your username</h2>
      <p style={{ fontSize: 13, color: "var(--ds-muted)", marginTop: 4 }}>
        Your public dashboard address —{" "}
        <span className="ds-mono" style={{ color: "var(--ds-dim)" }}>projects.sujaykumar.dev/xlearn/u/&lt;you&gt;</span>. You can
        also sign in with it.
      </p>

      <div style={{ marginTop: 20, display: "flex", flexDirection: "column", gap: 10 }}>
        <div className="ds-field">
          <label className="ds-field__label" htmlFor="onb-username">Username</label>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span className="ds-mono" style={{ color: "var(--ds-muted)" }}>/xlearn/u/</span>
            <input
              id="onb-username"
              className="ds-input"
              style={{ flex: 1 }}
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              placeholder="your-handle"
              aria-describedby="onb-username-hint"
              value={value}
              onChange={(e) => {
                if (setU.isError) setU.reset();
                setValue(e.target.value);
              }}
              onKeyDown={(e) => e.key === "Enter" && canClaim && primary()}
            />
          </div>
        </div>
        {/* Stable polite live region so screen readers hear availability/claim results. */}
        <div id="onb-username-hint" aria-live="polite">
          <UsernameStepHint value={normalized} changed={changed} current={currentUsername} avail={avail} setU={setU} settled={settled} />
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 7, fontSize: 11.5, color: "var(--ds-muted)" }}>
          <Icon name="eye" className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} /> Public, non-PII only — solved counts,
          streak and activity. Change it anytime in Settings.
        </div>
      </div>

      <button
        type="button"
        className="ds-btn ds-btn--primary ds-btn--block ds-btn--lg"
        style={{ marginTop: 20 }}
        disabled={setU.isPending || !(canClaim || (!!currentUsername && !changed))}
        onClick={primary}
      >
        {setU.isPending ? "Claiming…" : currentUsername && !changed ? "Continue" : "Claim & continue"}{" "}
        <Icon name="arrow" className="xl-ico--sm" />
      </button>
      <div style={{ display: "flex", gap: 10, marginTop: 10 }}>
        <button type="button" className="ds-btn ds-btn--ghost" onClick={onBack} disabled={setU.isPending}>
          Back
        </button>
        <button
          type="button"
          className="ds-btn ds-btn--ghost"
          style={{ flex: 1, justifyContent: "center" }}
          disabled={setU.isPending}
          onClick={onContinue}
        >
          Skip for now
        </button>
      </div>
    </>
  );
}

/** The live validity/availability line beneath the onboarding username field. */
function UsernameStepHint({
  value,
  changed,
  current,
  avail,
  setU,
  settled,
}: {
  value: string;
  changed: boolean;
  current?: string;
  avail: ReturnType<typeof useUsernameAvailability>;
  setU: ReturnType<typeof useSetUsername>;
  settled: boolean;
}) {
  if (setU.isError) return <span role="alert" style={{ fontSize: 12, color: "var(--ds-err)" }}>{setU.error.message || "Couldn’t claim that — try another."}</span>;
  if (value.length === 0) return <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>3–30 chars · lowercase letters, numbers, hyphens. Optional — you can skip this.</span>;
  if (!changed && current) return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>This is your current username.</span>;
  if (value.length < 3) return <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>Keep going — at least 3 characters.</span>;
  // Until the debounced probe catches up to the current input, show "checking" rather than a
  // stale availability result for the previous value.
  if (!settled || avail.isLoading) return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>Checking availability…</span>;
  if (avail.data?.available)
    return (
      <span style={{ fontSize: 12, color: "var(--ds-ok)", display: "inline-flex", alignItems: "center", gap: 5 }}>
        <Icon name="check" className="xl-ico--sm" /> @{value} is available
      </span>
    );
  if (avail.data) return <span style={{ fontSize: 12, color: "var(--ds-err)" }}>{avail.data.reason ?? "That username isn’t available."}</span>;
  return null;
}

function StepCoach({ onBack, onFinish }: { onBack: () => void; onFinish: () => void }) {
  const complete = useCompleteOnboarding();
  const putKey = usePutCoachKey();
  const [provider, setProvider] = useState<ProviderId>("anthropic");
  const [rawKey, setRawKey] = useState("");
  // The provider whose key was stored here, so a retry after a failed Finish doesn't
  // re-send it (the raw key is cleared from state once it is saved).
  const [savedFor, setSavedFor] = useState<ProviderId | null>(null);

  const key = rawKey.trim();
  const busy = putKey.isPending || complete.isPending;
  const meta = COACH_PROVIDERS.find((p) => p.id === provider)!;

  const completeOnboarding = () => complete.mutate(undefined, { onSuccess: onFinish });

  // Finish stores the typed key with the same PUT /coach/key Settings uses, and completes
  // onboarding only once the key is saved. A failed save keeps the learner here with the
  // error, to fix the key or skip. With no key typed, Finish just completes.
  const finish = () => {
    if (!key) {
      completeOnboarding();
      return;
    }
    const model = COACH_MODELS[provider][0]!.id;
    putKey.mutate(
      { provider, key, default_model: model, name: coachModelLabel(model) },
      {
        onSuccess: () => {
          setRawKey("");
          setSavedFor(provider);
          completeOnboarding();
        },
      },
    );
  };

  // Skip never stores the key, even if one was typed.
  const skip = () => {
    putKey.reset();
    completeOnboarding();
  };

  const editKey = (next: string) => {
    setRawKey(next);
    if (putKey.isError) putKey.reset();
  };

  return (
    <>
      <div className="xl-eyebrow">Step 4 of 4 · optional</div>
      <h2 style={{ fontSize: 22, fontWeight: 700, marginTop: 6 }}>Power up your coach</h2>
      <p style={{ fontSize: 13, color: "var(--ds-muted)", marginTop: 4 }}>
        Add an API key to enable the AI coach on every screen. Used only for your coach.
      </p>

      <div style={{ marginTop: 20, display: "flex", flexDirection: "column", gap: 12 }}>
        <div className="ds-field">
          <span className="ds-field__label">Provider</span>
          <div className="ds-seg" role="group" aria-label="Coach provider">
            {COACH_PROVIDERS.map((p) => (
              <button
                key={p.id}
                type="button"
                className={provider === p.id ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"}
                aria-pressed={provider === p.id}
                disabled={busy}
                onClick={() => {
                  setProvider(p.id);
                  if (putKey.isError) putKey.reset();
                }}
              >
                {p.label}
              </button>
            ))}
          </div>
        </div>
        <div className="ds-field">
          <label className="ds-field__label" htmlFor="onb-key">
            API key
          </label>
          <input
            id="onb-key"
            className="ds-input ds-input--mono"
            type="password"
            placeholder={savedFor === provider ? `${meta.label} key saved` : meta.keyHint}
            aria-label="API key"
            aria-invalid={putKey.isError || undefined}
            aria-describedby={putKey.isError ? "onb-key-error" : undefined}
            autoComplete="off"
            spellCheck={false}
            value={rawKey}
            disabled={busy}
            onChange={(e) => editKey(e.target.value)}
          />
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 7, fontSize: 11.5, color: "var(--ds-muted)" }}>
          <Icon name="lock" className="xl-ico--sm" style={{ color: "var(--ds-ok)" }} /> Stored encrypted with the coach — you can add it
          anytime from Settings.
        </div>
      </div>

      {putKey.isError && (
        <p id="onb-key-error" role="alert" style={{ color: "var(--ds-err)", fontSize: 12.5, marginTop: 12 }}>
          Couldn’t save your key{putKey.error.message ? ` (${putKey.error.message})` : ""}. Check it and try again, or skip for now and
          add it later in Settings.
        </p>
      )}
      {complete.isError && (
        <p role="alert" style={{ color: "var(--ds-err)", fontSize: 12.5, marginTop: 12 }}>
          {savedFor ? "Your key is saved, but we couldn’t finish setup. Please try again." : "Couldn’t finish setup. Please try again."}
        </p>
      )}

      <button
        type="button"
        className="ds-btn ds-btn--primary ds-btn--block ds-btn--lg"
        style={{ marginTop: 20 }}
        disabled={busy}
        onClick={finish}
      >
        <Icon name="spark" className="xl-ico--sm" />{" "}
        {putKey.isPending ? "Saving key…" : complete.isPending ? "Finishing…" : "Finish & enter xLearn"}
      </button>
      <div style={{ display: "flex", gap: 10, marginTop: 10 }}>
        <button type="button" className="ds-btn ds-btn--ghost" onClick={onBack} disabled={busy}>
          Back
        </button>
        <button type="button" className="ds-btn ds-btn--ghost" style={{ flex: 1, justifyContent: "center" }} disabled={busy} onClick={skip}>
          Skip for now
        </button>
      </div>
    </>
  );
}

function Centered({ children }: { children: string }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "24px 0" }}>
      <Spinner label={children} />
    </div>
  );
}

// --- brand marks ---

function GitHubMark() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true" style={{ marginRight: 2 }}>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
    </svg>
  );
}
