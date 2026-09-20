import { QueryClient } from "@tanstack/react-query";

// Shared TanStack Query client. The app is read-heavy over the BFF (ADR-0008);
// these defaults suit cache-friendly screen reads. Wired but calling nothing
// this sprint.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});
