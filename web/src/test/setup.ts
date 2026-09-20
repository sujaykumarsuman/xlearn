// Vitest setup: register jest-dom matchers on vitest's expect and reset the DOM
// between tests.
import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(() => {
  cleanup();
});
