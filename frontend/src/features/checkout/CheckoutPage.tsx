import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getOrder, payOrder } from "../../api/orders";
import { formatPrice } from "../../components/ui/Price";
import { useCart } from "../cart/cartStore";
import { useCountdown } from "./useCountdown";
import { usePageTitle } from "../../components/usePageTitle";

const SUCCESS_CARD = "4242 4242 4242 4242";
const DECLINE_CARD = "4000 0000 0000 0002";

/**
 * /checkout/:id: pay for a pending order before its 15-minute
 * reservation runs out. Payments are simulated (no real money).
 */
export function CheckoutPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const clearCart = useCart((s) => s.clear);
  const [card, setCard] = useState("");
  usePageTitle("Checkout");

  const order = useQuery({ queryKey: ["order", id], queryFn: () => getOrder(id) });
  const secondsLeft = useCountdown(order.data?.expires_at);

  const pay = useMutation({
    mutationFn: () => payOrder(id, card),
    onSuccess: (paid) => {
      queryClient.setQueryData(["order", id], paid);
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      clearCart();
      navigate(`/orders/${id}`, { replace: true });
    },
    // A decline or an expired reservation changes the order's status.
    onError: () => order.refetch(),
  });

  if (order.isError) {
    return <Message title="Order not found" text={order.error.message} />;
  }
  const o = order.data;
  if (!o) return <div className="mx-auto mt-20 size-10 animate-spin rounded-full border-4 border-line border-t-accent" />;
  if (o.status === "paid") return <Navigate to={`/orders/${o.id}`} replace />;
  if (o.status !== "pending" || secondsLeft === 0) {
    return (
      <Message
        title={o.status === "failed" ? "Payment declined" : "Reservation expired"}
        text={
          pay.error?.message ??
          "The keys for this order were released. Your cart is still saved, so you can check out again."
        }
      />
    );
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    pay.mutate();
  }

  const minutes = Math.floor(secondsLeft / 60);
  const seconds = String(secondsLeft % 60).padStart(2, "0");

  return (
    <div className="mx-auto grid max-w-4xl gap-8 md:grid-cols-[1fr_320px]">
      <section className="space-y-4">
        <h1 className="text-2xl font-bold">Checkout</h1>
        <p className="rounded-md bg-cta/15 px-4 py-3 text-sm">
          Your keys are reserved for{" "}
          <span className="font-bold tabular-nums" aria-live="polite">
            {minutes}:{seconds}
          </span>
          . Pay before the time runs out.
        </p>

        <form onSubmit={onSubmit} className="space-y-4 rounded-lg bg-surface p-6">
          <div>
            <label htmlFor="card" className="mb-1.5 block text-sm font-medium">
              Card number
            </label>
            <input
              id="card"
              inputMode="numeric"
              autoComplete="off"
              required
              value={card}
              onChange={(e) => setCard(e.target.value)}
              placeholder={SUCCESS_CARD}
              className="h-11 w-full rounded-md border border-line bg-page px-3 font-mono text-sm tracking-wider placeholder:text-muted/60 focus:border-accent focus:outline-none"
            />
          </div>
          <div className="text-xs text-muted">
            Demo payments, no real money. Test cards:{" "}
            <button type="button" onClick={() => setCard(SUCCESS_CARD)} className="text-accent underline">
              {SUCCESS_CARD}
            </button>{" "}
            (approved) or{" "}
            <button type="button" onClick={() => setCard(DECLINE_CARD)} className="text-accent underline">
              {DECLINE_CARD}
            </button>{" "}
            (declined).
          </div>
          {pay.isError && (
            <p role="alert" className="rounded-md border border-discount/50 bg-discount/10 px-3 py-2 text-sm">
              {pay.error.message}
            </p>
          )}
          <button
            type="submit"
            disabled={pay.isPending}
            className="h-12 w-full rounded-md bg-cta font-semibold text-page hover:bg-cta-hover disabled:opacity-60"
          >
            {pay.isPending ? "Paying…" : `Pay ${formatPrice(o.total_cents)}`}
          </button>
        </form>
      </section>

      <aside className="h-fit space-y-3 rounded-lg bg-surface p-5">
        <h2 className="font-bold">Your order</h2>
        <ul className="space-y-2 text-sm">
          {o.items.map((it) => (
            <li key={it.id} className="flex justify-between gap-3">
              <span className="min-w-0">
                <span className="line-clamp-2">{it.product.name}</span>
                <span className="text-xs text-muted">
                  {it.quantity} × {formatPrice(it.unit_price_cents)} · {it.seller_name}
                </span>
              </span>
              <span className="shrink-0 font-semibold">{formatPrice(it.unit_price_cents * it.quantity)}</span>
            </li>
          ))}
        </ul>
        <div className="flex justify-between border-t border-line pt-3 font-bold">
          <span>Total</span>
          <span>{formatPrice(o.total_cents)}</span>
        </div>
      </aside>
    </div>
  );
}

function Message({ title, text }: { title: string; text: string }) {
  return (
    <div className="py-20 text-center">
      <h1 className="text-2xl font-bold">{title}</h1>
      <p className="mx-auto mt-2 max-w-md text-muted">{text}</p>
      <Link to="/cart" className="mt-6 inline-block rounded-md bg-cta px-5 py-2 font-semibold text-page hover:bg-cta-hover">
        Back to cart
      </Link>
    </div>
  );
}
