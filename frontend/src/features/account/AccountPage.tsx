import { useCurrentUser } from "../auth/useCurrentUser";
import { useLogout } from "../auth/useLogout";

/** /account: only reachable when signed in (see RequireAuth in App.tsx). */
export function AccountPage() {
  const { user } = useCurrentUser();
  const signOut = useLogout();
  if (!user) return null; // RequireAuth makes sure this doesn't happen

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-3xl font-bold">My account</h1>

      <section className="rounded-lg bg-surface p-6">
        <dl className="grid grid-cols-[auto_1fr] gap-x-8 gap-y-3 text-sm">
          <dt className="text-muted">Name</dt>
          <dd className="font-semibold">{user.display_name}</dd>
          <dt className="text-muted">Email</dt>
          <dd>{user.email}</dd>
          <dt className="text-muted">Member since</dt>
          <dd>{new Date(user.created_at).toLocaleDateString("en-GB", { dateStyle: "long" })}</dd>
        </dl>
      </section>

      <section className="rounded-lg bg-surface p-6">
        <h2 className="mb-2 text-lg font-bold">My orders</h2>
        <p className="text-sm text-muted">Your orders and their keys will appear here once checkout is built (day 5).</p>
      </section>

      <button
        onClick={() => signOut.mutate()}
        disabled={signOut.isPending}
        className="rounded-md border border-line px-5 py-2.5 text-sm font-semibold hover:bg-surface disabled:opacity-60"
      >
        {signOut.isPending ? "Signing out…" : "Sign out"}
      </button>
    </div>
  );
}
