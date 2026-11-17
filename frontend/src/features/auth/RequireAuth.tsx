import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router";
import { useCurrentUser } from "./useCurrentUser";

/**
 * Wraps pages that need a signed-in user:
 *   <Route path="account" element={<RequireAuth><AccountPage /></RequireAuth>} />
 * Signed out -> /login?next=<this page>, so after signing in you come back.
 *
 * This only hides pages; the real protection is on the server, where every
 * private endpoint sits behind RequireUser and answers 401 without a session.
 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { user, isLoading } = useCurrentUser();
  const location = useLocation();

  if (isLoading) {
    return <div className="mx-auto mt-20 size-10 animate-spin rounded-full border-4 border-line border-t-accent" />;
  }
  if (!user) {
    const here = location.pathname + location.search;
    return <Navigate to={`/login?next=${encodeURIComponent(here)}`} replace />;
  }
  return children;
}
