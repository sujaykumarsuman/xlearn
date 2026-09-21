import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmptyState, ErrorState, Spinner } from "./States";
import { IconSprite } from "./Icon";

describe("shared state components", () => {
  it("Spinner is a status region whose rotation is on a real glyph (not an empty span)", () => {
    const { container } = render(
      <>
        <IconSprite />
        <Spinner label="Loading paths…" />
      </>,
    );
    expect(screen.getByRole("status")).toHaveTextContent("Loading paths…");
    // The DS `.ds-spin` animation is applied to an <svg> glyph, so the spinner is
    // actually visible (the old bug was `.ds-spin` on an empty <span>).
    expect(container.querySelector("svg.ds-spin")).not.toBeNull();
  });

  it("ErrorState is an alert with a working Retry", () => {
    const onRetry = vi.fn();
    render(<ErrorState message="Couldn’t load the catalog." onRetry={onRetry} />);
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn’t load the catalog.");
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("ErrorState omits Retry when no handler is provided", () => {
    render(<ErrorState message="x" />);
    expect(screen.queryByRole("button", { name: /retry/i })).toBeNull();
  });

  it("EmptyState announces its title and supporting copy", () => {
    render(<EmptyState title="No learning paths yet">Check back soon.</EmptyState>);
    expect(screen.getByRole("status")).toHaveTextContent("No learning paths yet");
    expect(screen.getByText(/check back soon/i)).toBeInTheDocument();
  });
});
