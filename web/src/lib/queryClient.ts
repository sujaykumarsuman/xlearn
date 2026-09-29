import { QueryClient } from "@tanstack/react-query";
import { queryRetryDelay, shouldRetryQuery } from "./api";

// Shared TanStack Query client. The app is read-heavy over the BFF (ADR-0008);
// these defaults suit cache-friendly screen reads. A 429 is never retried sooner than its
// Retry-After, and a long one isn't auto-retried at all (m1-05; api.ts shouldRetryQuery).
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: shouldRetryQuery,
      retryDelay: queryRetryDelay,
      refetchOnWindowFocus: false,
    },
  },
});
