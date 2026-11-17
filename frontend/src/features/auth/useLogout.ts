import { useMutation } from "@tanstack/react-query";
import { logout } from "../../api/auth";

/**
 * Sign out: the server deletes the session, then we do a full page load of
 * /store instead of a normal in-app navigation. A fresh page load throws away
 * everything cached in memory, so no private data (account details, later
 * orders and keys) survives the sign-out.
 *
 * (Updating the cache and navigating in-app raced: the account page's
 * RequireAuth saw "no user" first and redirected to /login instead.)
 */
export function useLogout() {
  return useMutation({
    mutationFn: logout,
    onSuccess: () => window.location.assign("/store"),
  });
}
