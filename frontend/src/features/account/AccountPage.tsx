import { Link } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { listOrders } from "../../api/orders";
import { formatPrice } from "../../components/ui/Price";
import { useCurrentUser } from "../auth/useCurrentUser";
import { StatusBadge } from "../orders/StatusBadge";
import { useLogout } from "../auth/useLogout";
import { usePageTitle } from "../../components/usePageTitle";

/** /account: only reachable when signed in (see RequireAuth in App.tsx). */
export function AccountPage() {
  const { user } = useCurrentUser();
  usePageTitle("My account");
  const signOut = useLogout();
  const orders = useQuery({ queryKey: ["orders"], queryFn: listOrders });
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
        <h2 className="mb-4 text-lg font-bold">My orders</h2>
        {orders.isError ? (
          <p className="text-sm text-muted">Couldn't load your orders: {orders.error.message}</p>
        ) : !orders.data ? (
          <div className="h-16 animate-pulse rounded bg-surface-hover" />
        ) : orders.data.length === 0 ? (
          <p className="text-sm text-muted">
            No orders yet. <Link to="/store" className="text-accent hover:underline">Find something in the store</Link>.
          </p>
        ) : (
          <ul className="divide-y divide-line">
            {orders.data.map((o) => (
              <li key={o.id}>
                <Link to={`/orders/${o.id}`} className="flex items-center gap-4 py-3 hover:text-accent">
                  <span className="w-28 shrink-0 text-sm text-muted">
                    {new Date(o.created_at).toLocaleDateString("en-GB", { dateStyle: "medium" })}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-sm">
                    {o.items.map((it) => it.product.title).join(", ")}
                  </span>
                  <StatusBadge status={o.status} />
                  <span className="w-20 text-right font-semibold">{formatPrice(o.total_cents)}</span>
                </Link>
              </li>
            ))}
          </ul>
        )}
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
