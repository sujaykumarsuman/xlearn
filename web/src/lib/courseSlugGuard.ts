// The course-slug guard, SPA side (ADR-0026 §5; AB02 "Routes and reserved segments").
// A course lives at /xlearn/:course/…, one dynamic first segment, so no course slug may
// equal a static top-level SPA segment (u, auth, settings) or a gateway-owned path (api,
// assets, healthz, readyz, privacy). The list is generated from the Go guard
// (internal/course, ReservedSegments) into reservedSegments.json, and a parity test pins
// this module to that file, so the Go guard, the JSON and the SPA agree. Any new static
// top-level SPA route joins the Go list first.
import reserved from "./reservedSegments.json";

/** The reserved first path segments: never a course slug. */
export const RESERVED_SEGMENTS: readonly string[] = Object.freeze([...reserved.reserved]);

const RESERVED = new Set(RESERVED_SEGMENTS);

/** isReservedSegment reports whether a first path segment is reserved (not a course). */
export function isReservedSegment(seg: string): boolean {
  return RESERVED.has(seg);
}
