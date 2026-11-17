import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "./api/client";
import { App } from "./app/App";
import "./styles/index.css";

// TanStack Query caches server data by key, dedupes identical requests and
// handles loading/error state, so components don't need useEffect + fetch.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000, // reuse a result for 30s before refetching
      // Retry once on network/server errors, but never on 4xx: a 404 or a
      // validation error will give the same answer every time.
      retry: (failureCount, error) =>
        failureCount < 1 && !(error instanceof ApiError && error.status < 500),
      refetchOnWindowFocus: false,
    },
  },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
