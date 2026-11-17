import { useMemo } from "react";
import { Link, useNavigate } from "react-router";
import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";
import { createOrder, quoteCart } from "../../api/orders";
import type { QuoteLine } from "../../api/types";
import { Cover } from "../../components/ui/Cover";
import { formatPrice } from "../../components/ui/Price";
import { useCurrentUser } from "../auth/useCurrentUser";
import { MAX_QTY, useCart, type CartLine } from "./cartStore";

/**
 * /cart: the lines come from the browser, but every price, stock level and
 * the total come from POST /api/cart/quote, so what you see is what the
 * server will charge. Checkout is only enabled when every line is buyable.
 */
export function CartPage() {
  const lines = useCart((s) => s.lines);
  const { user } = useCurrentUser();
  const navigate = useNavigate();

  const items = lines.map((l) => ({ offer_id: l.offer_id, qty: l.qty }));
  const itemsKey = JSON.stringify(items);

  const quote = useQuery({
    queryKey: ["quote", itemsKey],
    queryFn: () => quoteCart(items),
    enabled: items.length > 0,
    placeholderData: keepPreviousData, // keep showing totals while a quantity change reloads
  });

  // One Idempotency-Key per cart content: a double click on "Checkout"
  // sends the same key twice, and the server creates only one order.
  const idempotencyKey = useMemo(() => crypto.randomUUID(), [itemsKey]);

  const checkout = useMutation({
    mutationFn: () => createOrder(items, idempotencyKey),
    onSuccess: (order) => navigate(`/checkout/${order.id}`),
    onError: () => quote.refetch(), // e.g. out_of_stock: show which line changed
  });

  if (lines.length === 0) {
    return (
      <div className="py-24 text-center">
        <h1 className="text-2xl font-bold">Your cart is empty</h1>
        <p className="mt-2 text-muted">Find something you like and add it from the product page.</p>
        <Link to="/store" className="mt-6 inline-block rounded-md bg-cta px-5 py-2 font-semibold text-page hover:bg-cta-hover">
          Browse the store
        </Link>
      </div>
    );
  }

  const q = quote.data;
  const canCheckout = !!q && q.all_available && !quote.isFetching && !checkout.isPending;

  return (
    <div className="grid gap-8 lg:grid-cols-[1fr_340px]">
      <section>
        <h1 className="mb-4 text-2xl font-bold">Cart</h1>
        <ul className="space-y-3">
          {lines.map((line) => (
            <CartRow key={line.offer_id} line={line} quoted={q?.items.find((i) => i.offer_id === line.offer_id)} />
          ))}
        </ul>
      </section>

      <aside className="h-fit rounded-lg bg-surface p-5 lg:sticky lg:top-24">
        <h2 className="mb-4 text-lg font-bold">Order summary</h2>
        <div className="flex items-baseline justify-between">
          <span className="text-muted">Total</span>
          <span className="text-2xl font-bold">{q ? formatPrice(q.total_cents) : "…"}</span>
        </div>
        {q && !q.all_available && (
          <p className="mt-3 text-sm text-discount">Some items changed. Update or remove them to continue.</p>
        )}
        {checkout.isError && (
          <p role="alert" className="mt-3 text-sm text-discount">
            {checkout.error.message}
          </p>
        )}

        {user ? (
          <button
            onClick={() => checkout.mutate()}
            disabled={!canCheckout}
            className="mt-5 h-12 w-full rounded-md bg-cta font-semibold text-page hover:bg-cta-hover disabled:cursor-not-allowed disabled:opacity-60"
          >
            {checkout.isPending ? "Reserving your keys…" : "Checkout"}
          </button>
        ) : (
          <Link
            to="/login?next=%2Fcart"
            className="mt-5 block h-12 rounded-md bg-cta text-center font-semibold leading-[3rem] text-page hover:bg-cta-hover"
          >
            Sign in to check out
          </Link>
        )}
        <p className="mt-3 text-xs text-muted">Keys are reserved for 15 minutes while you pay.</p>
      </aside>
    </div>
  );
}

/** One cart line: what's in the browser, corrected by the server's quote. */
function CartRow({ line, quoted }: { line: CartLine; quoted: QuoteLine | undefined }) {
  const setQty = useCart((s) => s.setQty);
  const remove = useCart((s) => s.remove);

  const price = quoted?.price_cents ?? line.price_cents;
  const maxQty = Math.min(MAX_QTY, quoted?.stock || MAX_QTY);
  const priceChanged = quoted?.status === "ok" && quoted.price_cents !== line.price_cents;

  return (
    <li className="flex gap-4 rounded-lg bg-surface p-4">
      <Link to={`/product/${line.slug}`} className="shrink-0">
        <Cover title={line.title} imageUrl={quoted?.product?.image_url ?? null} showTitle={false} className="h-24 w-[4.5rem] rounded" />
      </Link>

      <div className="min-w-0 flex-1">
        <Link to={`/product/${line.slug}`} className="line-clamp-2 font-semibold hover:text-accent">
          {line.name}
        </Link>
        <p className="text-sm text-muted">Sold by {line.seller_name}</p>

        {/* What the quote says about this line */}
        {quoted?.status === "sold_out" && <p className="mt-1 text-sm text-discount">Sold out: please remove it.</p>}
        {quoted?.status === "unavailable" && <p className="mt-1 text-sm text-discount">No longer available: please remove it.</p>}
        {quoted?.status === "not_enough_stock" && (
          <p className="mt-1 text-sm text-discount">
            Only {quoted.stock} left.{" "}
            <button onClick={() => setQty(line.offer_id, quoted.stock)} className="underline">
              Change to {quoted.stock}
            </button>
          </p>
        )}
        {priceChanged && (
          <p className="mt-1 text-sm text-cta">Price changed from {formatPrice(line.price_cents)}</p>
        )}

        <div className="mt-3 flex items-center gap-3">
          <div className="flex items-center rounded-md border border-line">
            <button
              onClick={() => setQty(line.offer_id, line.qty - 1)}
              disabled={line.qty <= 1}
              aria-label="One less"
              className="size-8 disabled:opacity-40"
            >
              −
            </button>
            <span className="w-8 text-center text-sm" aria-label="Quantity">{line.qty}</span>
            <button
              onClick={() => setQty(line.offer_id, line.qty + 1)}
              disabled={line.qty >= maxQty}
              aria-label="One more"
              className="size-8 disabled:opacity-40"
            >
              +
            </button>
          </div>
          <button onClick={() => remove(line.offer_id)} className="text-sm text-muted hover:text-white">
            Remove
          </button>
        </div>
      </div>

      <div className="text-right">
        <p className="text-lg font-bold">{formatPrice(price * line.qty)}</p>
        {line.qty > 1 && <p className="text-xs text-muted">{formatPrice(price)} each</p>}
      </div>
    </li>
  );
}
