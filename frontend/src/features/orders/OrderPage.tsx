import { useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { getOrder } from "../../api/orders";
import { formatPrice } from "../../components/ui/Price";
import { StatusBadge } from "./StatusBadge";
import { usePageTitle } from "../../components/usePageTitle";

/** /orders/:id: one order. Once it's paid, this is where the keys are. */
export function OrderPage() {
  const { id = "" } = useParams();
  usePageTitle("Order");
  const order = useQuery({ queryKey: ["order", id], queryFn: () => getOrder(id) });

  if (order.isError) {
    return (
      <div className="py-20 text-center">
        <h1 className="text-2xl font-bold">Order not found</h1>
        <Link to="/account" className="mt-4 inline-block text-accent hover:underline">My account</Link>
      </div>
    );
  }
  const o = order.data;
  if (!o) return <div className="mx-auto mt-20 size-10 animate-spin rounded-full border-4 border-line border-t-accent" />;

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <Link to="/account" className="text-sm text-muted hover:text-white">‹ My orders</Link>
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-2xl font-bold">Order {o.id.slice(0, 8)}</h1>
        <StatusBadge status={o.status} />
      </div>
      <p className="text-sm text-muted">
        Placed {new Date(o.created_at).toLocaleString("en-GB", { dateStyle: "long", timeStyle: "short" })}
        {o.payment_ref && <> · payment {o.payment_ref}</>}
      </p>

      {o.status === "paid" && (
        <p className="rounded-md bg-region/15 px-4 py-3 text-sm">
          Payment received. Your keys are below; redeem each one on the platform shown.
        </p>
      )}
      {o.status === "pending" && (
        <Link to={`/checkout/${o.id}`} className="inline-block rounded-md bg-cta px-5 py-2 font-semibold text-page">
          Complete payment
        </Link>
      )}

      <ul className="space-y-3">
        {o.items.map((it) => (
          <li key={it.id} className="rounded-lg bg-surface p-5">
            <div className="flex justify-between gap-4">
              <div>
                <Link to={`/product/${it.product.slug}`} className="font-semibold hover:text-accent">{it.product.name}</Link>
                <p className="text-sm text-muted">
                  {it.quantity} × {formatPrice(it.unit_price_cents)} · sold by {it.seller_name}
                </p>
              </div>
              <span className="font-bold">{formatPrice(it.unit_price_cents * it.quantity)}</span>
            </div>
            {it.keys.length > 0 && (
              <ul className="mt-4 space-y-2">
                {it.keys.map((k) => (
                  <KeyRow key={k} code={k} platform={it.product.platform} />
                ))}
              </ul>
            )}
          </li>
        ))}
      </ul>

      <p className="text-right text-lg">
        Total <span className="ml-2 text-2xl font-bold">{formatPrice(o.total_cents)}</span>
      </p>
    </div>
  );
}

function KeyRow({ code, platform }: { code: string; platform: string }) {
  const [copied, setCopied] = useState(false);
  async function copy() {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }
  return (
    <li className="flex items-center justify-between gap-3 rounded-md border border-dashed border-accent/60 bg-page px-4 py-2.5">
      <span>
        <span className="font-mono text-lg tracking-widest">{code}</span>
        <span className="ml-3 text-xs text-muted">{platform}</span>
      </span>
      <button onClick={copy} className="rounded bg-surface-hover px-3 py-1 text-sm hover:bg-line">
        {copied ? "Copied ✓" : "Copy"}
      </button>
    </li>
  );
}
