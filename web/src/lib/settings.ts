// Settings data hooks for the BFF (docs/architecture/api.md). The coach-key read is
// the S10 shell: GET /coach/key returns the masked key(s) + connected flag (never a raw
// key). When coach is not deployed yet (S11) the gateway returns the empty state
// ({keys:[], connected:false}) so the API-keys section renders "No key — coach off".
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";

/** One provider key as the coach service masks it (never the raw secret). */
export interface CoachKey {
  provider: string;
  masked_key: string;
  default_model: string;
  enabled: boolean;
  tested?: boolean;
}

/** GET /coach/key payload. `connected` is true when at least one key is enabled. */
export interface CoachKeyResponse {
  keys: CoachKey[];
  connected: boolean;
}

/** useCoachKey fetches the masked coach-key config. Retry is off so the empty state
 *  renders promptly if coach is unavailable. */
export function useCoachKey() {
  return useQuery<CoachKeyResponse, ApiRequestError>({
    queryKey: ["coach-key"],
    queryFn: () => apiFetch<CoachKeyResponse>("/coach/key"),
    retry: false,
    staleTime: 30_000,
  });
}
