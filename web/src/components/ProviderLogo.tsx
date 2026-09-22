/** ProviderLogo renders a provider's brand mark (nominative use) inside a rounded tile.
 *  Shared by the Settings coach panels and the header coach quick-switch. `bare` renders
 *  just the glyph (no tile) — for the compact pill switch. */
export function ProviderLogo({ provider, size = 28, bare = false }: { provider: string; size?: number; bare?: boolean }) {
  const glyph =
    provider === "openai" ? (
      <svg width={size * (bare ? 1 : 0.62)} height={size * (bare ? 1 : 0.62)} viewBox="0 0 24 24" fill={bare ? "currentColor" : "#fff"} aria-hidden="true">
        <path d="M22.28 9.82a5.98 5.98 0 0 0-.52-4.91 6.05 6.05 0 0 0-6.5-2.9A5.98 5.98 0 0 0 10.72.02 6.05 6.05 0 0 0 4.98 4.2 5.98 5.98 0 0 0 1 7.09a6.05 6.05 0 0 0 .74 7.1 5.98 5.98 0 0 0 .52 4.9 6.05 6.05 0 0 0 6.51 2.9A5.98 5.98 0 0 0 13.28 24a6.05 6.05 0 0 0 5.76-4.19 5.98 5.98 0 0 0 3.98-2.9 6.05 6.05 0 0 0-.74-7.1Zm-9 12.6a4.48 4.48 0 0 1-2.88-1.04l.14-.08 4.78-2.76a.78.78 0 0 0 .39-.68v-6.74l2.02 1.17a.07.07 0 0 1 .04.05v5.58a4.5 4.5 0 0 1-4.5 4.5ZM3.6 18.1a4.48 4.48 0 0 1-.54-3.02l.14.09 4.78 2.76a.78.78 0 0 0 .78 0l5.84-3.37v2.33a.07.07 0 0 1-.03.06L9.73 21.8a4.5 4.5 0 0 1-6.14-1.65Zm-1.26-10.4a4.48 4.48 0 0 1 2.35-1.97v5.68a.78.78 0 0 0 .39.68l5.84 3.37-2.02 1.17a.07.07 0 0 1-.07 0L4.4 19.4a4.5 4.5 0 0 1-2.06-6.02Zm16.6 3.86-5.84-3.38 2.02-1.16a.07.07 0 0 1 .07 0l4.83 2.79a4.5 4.5 0 0 1-.68 8.12v-5.69a.78.78 0 0 0-.4-.68Zm2.01-3.02-.14-.09-4.78-2.76a.78.78 0 0 0-.78 0L9.4 9.02V6.7a.07.07 0 0 1 .03-.06l4.83-2.79a4.5 4.5 0 0 1 6.68 4.67ZM8.3 12.86l-2.02-1.17a.07.07 0 0 1-.04-.05V6.06a4.5 4.5 0 0 1 7.38-3.45l-.14.08L8.7 5.45a.78.78 0 0 0-.39.68l-.01 6.73Zm1.1-2.36L12 8.99l2.6 1.5v3l-2.6 1.5-2.6-1.5v-3Z" />
      </svg>
    ) : (
      <svg width={size * (bare ? 0.95 : 0.6)} height={size * (bare ? 0.66 : 0.42)} viewBox="0 0 46 32" fill="#D97757" aria-hidden="true">
        <path d="M32.73 0h-6.3l11.16 32h6.3L32.73 0ZM13.27 0 2.1 32h6.42l2.28-6.66h11.68L24.77 32h6.42L20.02 0h-6.75Zm-.38 19.2 3.82-11.16L20.55 19.2h-7.66Z" />
      </svg>
    );
  if (bare) return glyph;
  const bg = provider === "openai" ? "#0a0a0a" : "#141413";
  return (
    <span
      className="xl-logo"
      style={{ width: size, height: size, background: bg, border: "1px solid var(--ds-line-2)" }}
      aria-label={provider === "openai" ? "OpenAI" : "Anthropic"}
    >
      {glyph}
    </span>
  );
}
