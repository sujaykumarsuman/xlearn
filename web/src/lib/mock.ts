// Mock-interview data hooks for the BFF (docs/architecture/api.md, Mock & progress).
// A mock is a 45-minute, server-timed session (POST /mocks -> GET /mocks/{id}); the
// phase rail + remaining time are server-authoritative (the client mirrors them). It is
// scored on the 7-dimension rubric (POST /mocks/{id}/score -> /35) and charted vs the
// readiness targets (GET /mocks/trend).
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";

export type Difficulty = "easy" | "med" | "hard";

/** The curriculum problem metadata the gateway enriches a mock with (absent when
 *  curriculum can't resolve the id — the screen falls back to the bare id). */
export interface MockProblem {
  id: string;
  title: string;
  difficulty: Difficulty;
  pattern: string;
  week_n: number;
}

/** One phase in the 45-minute rail (R-MK1), with its server-computed state. */
export interface MockPhase {
  index: number;
  label: string;
  range: string;
  startMin: number;
  endMin: number;
  grow: number;
  prompt: string;
  state: "past" | "current" | "upcoming";
}

/** The server-authoritative timer + phase state (R-MK1). */
export interface MockRail {
  elapsedSeconds: number;
  remainingSeconds: number;
  overtime: boolean;
  phaseIndex: number;
  phaseLabel: string;
  phases: MockPhase[];
}

/** One scored rubric dimension (key + display name + 1..5). */
export interface MockDimension {
  key: string;
  name: string;
  score: number;
}

/** The R-MK3 readiness targets (dashed lines on the trend). */
export interface MockTargets {
  w13: number;
  w15: number;
  pre: number;
}

/** GET /mocks/{id} (and POST /mocks, POST /mocks/{id}/score) response. */
export interface MockSession {
  id: string;
  status: "live" | "scored";
  setId: string;
  problemId: string;
  difficulty: Difficulty;
  date: string;
  startedAt: string;
  deadlineAt: string;
  total35: number | null;
  notes: string;
  rail: MockRail;
  dimensions: MockDimension[];
  targets: MockTargets;
  problem?: MockProblem | null;
}

/** The setup selection POST /mocks accepts. */
export interface MockSetup {
  setId: string;
  problemId?: string;
  difficulty: Difficulty;
}

/** A rubric submission: dimension key -> integer score in [1,5]. */
export type RubricScores = Record<string, number>;

/** One scored mock in the trend series (R-MK3). */
export interface MockTrendPoint {
  mockId: string;
  setId: string;
  problemId: string;
  difficulty: string;
  date: string;
  startedAt: string;
  total35: number;
}

/** GET /mocks/trend response. */
export interface MockTrend {
  points: MockTrendPoint[];
  targets: MockTargets;
}

/** The seven rubric dimensions in canonical PRD order (R-MK2), mirrored from the
 *  server so the scoring form + radar axes render in a stable order. */
export const RUBRIC_DIMENSIONS: ReadonlyArray<{ key: string; name: string }> = [
  { key: "communication", name: "Communication" },
  { key: "problem_understanding", name: "Problem understanding" },
  { key: "brute_force", name: "Brute force" },
  { key: "optimisation", name: "Optimisation" },
  { key: "code_quality", name: "Code quality" },
  { key: "edge_cases", name: "Edge cases" },
  { key: "complexity", name: "Complexity" },
];

/** The full 45-minute window in seconds (R-MK1). */
export const MOCK_TOTAL_SECS = 45 * 60;

/** useMock fetches a session and polls it every 5s while it is live so the phase rail
 *  and remaining time stay server-driven (polling stops once the session is scored). */
export function useMock(id: string) {
  return useQuery<MockSession, ApiRequestError>({
    queryKey: ["mock", id],
    queryFn: () => apiFetch<MockSession>(`/mocks/${encodeURIComponent(id)}`),
    enabled: id !== "",
    refetchInterval: (query) => (query.state.data?.status === "scored" ? false : 5000),
  });
}

/** useMockTrend fetches the account's scored /35 history vs the readiness targets. */
export function useMockTrend(enabled = true) {
  return useQuery<MockTrend, ApiRequestError>({
    queryKey: ["mock", "trend"],
    queryFn: () => apiFetch<MockTrend>("/mocks/trend"),
    enabled,
  });
}

/** startMock begins a 45-minute session (POST /mocks) and returns the live session. */
export function startMock(setup: MockSetup): Promise<MockSession> {
  return apiFetch<MockSession>("/mocks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(setup),
  });
}

/** scoreMock submits the seven-dimension rubric (POST /mocks/{id}/score -> /35). */
export function scoreMock(id: string, scores: RubricScores, notes: string): Promise<MockSession> {
  return apiFetch<MockSession>(`/mocks/${encodeURIComponent(id)}/score`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scores, notes }),
  });
}
