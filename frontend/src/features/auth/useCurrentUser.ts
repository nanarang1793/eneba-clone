import { useQuery } from "@tanstack/react-query";
import { getMe } from "../../api/auth";

// One cache entry, ["me"], holds who is signed in. Login/register/logout
// write to it directly (queryClient.setQueryData), so the header and every
// protected page update at once without another request.
export const ME_KEY = ["me"] as const;

export function useCurrentUser() {
  const me = useQuery({ queryKey: ME_KEY, queryFn: getMe, staleTime: 5 * 60_000 });
  return { user: me.data ?? null, isLoading: me.isPending };
}

/**
 * Where to go after signing in. Only same-site paths are allowed:
 * "/account" is fine, but "//evil.com", "/\evil.com" (browsers read the
 * backslash as a slash) or "https://evil.com" would turn the login page
 * into a redirect to another website.
 */
export function safeNext(next: string | null): string {
  if (next && next.startsWith("/") && !next.startsWith("//") && !next.startsWith("/\\")) return next;
  return "/account";
}
