# C4 L1 — System context

```mermaid
graph TB
    learner["👤 Learner<br/>SWE prepping for interviews"]
    sys["xLearn<br/>guided-learning platform<br/>projects.sujaykumar.dev/xlearn"]
    github["GitHub OAuth"]
    google["Google OAuth"]
    llm["LLM provider<br/>OpenAI / Anthropic<br/>(user's own API key)"]
    ext["LeetCode / NeetCode<br/>(outbound deep links)"]

    learner -->|browser| sys
    sys -->|OIDC login| github
    sys -->|OIDC login| google
    sys -->|coach calls, user's key| llm
    learner -.->|opens problem links| ext
```

- xLearn owns the **method** (gating, spaced repetition, mistakes, mocks); it links out to problem
  hosts rather than embedding them.
- The only outbound integrations are **OAuth** (login) and the **LLM provider** (coach, with the
  user's own key — xLearn never provides inference).
