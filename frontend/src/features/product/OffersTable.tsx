import type { Offer } from "../../api/types";
import { formatPrice } from "../../components/ui/Price";

const compact = new Intl.NumberFormat("en", { notation: "compact" }); // 184220 -> "184K"

/**
 * Every seller's offer for one product. The backend already sorts them:
 * in-stock first, cheapest first, better-rated seller on equal prices.
 * The first in-stock row is flagged is_best and highlighted, like Eneba's
 * "best price" offer.
 */
export function OffersTable({ offers }: { offers: Offer[] }) {
  if (offers.length === 0) {
    return <p className="rounded-lg bg-surface p-6 text-center text-muted">No sellers are offering this product right now.</p>;
  }

  return (
    <ul className="space-y-2">
      {offers.map((o) => (
        <li
          key={o.id}
          className={`grid grid-cols-[1fr_auto] items-center gap-x-4 gap-y-2 rounded-lg border p-4 sm:grid-cols-[1fr_8rem_7rem_9rem] ${
            o.is_best ? "border-cta bg-surface-hover" : "border-transparent bg-surface"
          } ${o.in_stock ? "" : "opacity-50"}`}
        >
          {/* Seller */}
          <div>
            <p className="flex items-center gap-2 font-semibold">
              {o.seller.name}
              {o.is_best && (
                <span className="rounded bg-cta px-1.5 py-0.5 text-[10px] font-bold uppercase text-page">Best price</span>
              )}
            </p>
            <p className="text-xs text-muted">
              <span className="text-cta">★</span> {o.seller.rating.toFixed(2)} · {compact.format(o.seller.sales_count)} sales
            </p>
          </div>

          {/* Stock */}
          <p className={`text-sm ${o.in_stock ? "text-region" : "text-muted"} max-sm:order-last`}>
            {o.in_stock ? `In stock (${o.stock})` : "Out of stock"}
          </p>

          {/* Price */}
          <p className="text-right text-xl font-bold sm:text-left">{formatPrice(o.price_cents)}</p>

          {/* Buy: the cart and checkout arrive on day 5 */}
          <button
            disabled
            title="Checkout is coming soon"
            className="h-10 rounded-md bg-cta px-4 text-sm font-semibold text-page disabled:cursor-not-allowed disabled:opacity-60 max-sm:order-last"
          >
            {o.in_stock ? "Add to cart" : "Sold out"}
          </button>
        </li>
      ))}
    </ul>
  );
}
