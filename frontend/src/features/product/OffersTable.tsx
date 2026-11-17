import { useState } from "react";
import { Link } from "react-router";
import type { Offer, ProductDetail } from "../../api/types";
import { formatPrice } from "../../components/ui/Price";
import { MAX_QTY, useCart } from "../cart/cartStore";

const compact = new Intl.NumberFormat("en", { notation: "compact" }); // 184220 -> "184K"

/**
 * Every seller's offer for one product. The backend already sorts them:
 * in-stock first, cheapest first, better-rated seller on equal prices.
 * The first in-stock row is flagged is_best and highlighted, like Eneba's
 * "best price" offer.
 */
export function OffersTable({ offers, product }: { offers: Offer[]; product: ProductDetail }) {
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

          <BuyButton offer={o} product={product} />
        </li>
      ))}
    </ul>
  );
}

/** "Add to cart" for one offer. Can't add more than the seller has, or more than MAX_QTY. */
function BuyButton({ offer, product }: { offer: Offer; product: ProductDetail }) {
  const add = useCart((s) => s.add);
  const inCart = useCart((s) => s.lines.find((l) => l.offer_id === offer.id)?.qty ?? 0);
  const [justAdded, setJustAdded] = useState(false);

  const full = inCart >= Math.min(offer.stock, MAX_QTY);

  function onClick() {
    add({
      offer_id: offer.id,
      slug: product.slug,
      name: product.name,
      title: product.title,
      seller_name: offer.seller.name,
      price_cents: offer.price_cents,
    });
    setJustAdded(true);
    setTimeout(() => setJustAdded(false), 1500); // brief "Added ✓" feedback
  }

  let label = "Add to cart";
  if (!offer.in_stock) label = "Sold out";
  else if (justAdded) label = "Added ✓";
  else if (full) label = "Max in cart";

  return (
    <div className="flex flex-col items-stretch gap-1 max-sm:order-last">
      <button
        onClick={onClick}
        disabled={!offer.in_stock || full}
        className="h-10 rounded-md bg-cta px-4 text-sm font-semibold text-page hover:bg-cta-hover disabled:cursor-not-allowed disabled:opacity-60"
      >
        {label}
      </button>
      {inCart > 0 && (
        <Link to="/cart" className="text-center text-xs text-accent hover:underline">
          {inCart} in cart
        </Link>
      )}
    </div>
  );
}
