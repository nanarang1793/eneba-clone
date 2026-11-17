import { Link } from "react-router";
import type { Product } from "../../api/types";
import { Cover } from "../../components/ui/Cover";
import { formatPrice } from "../../components/ui/Price";

/** One product tile in the grid, laid out like Eneba's cards. Clicking opens the product page. */
export function ProductCard({ product }: { product: Product }) {
  const p = product;
  return (
    <Link
      to={`/product/${p.slug}`}
      className="group flex flex-col overflow-hidden rounded-lg bg-surface transition hover:-translate-y-0.5 hover:bg-surface-hover focus-visible:outline-2 focus-visible:outline-accent"
    >
      <div className="relative aspect-[3/4]">
        <Cover title={p.title} imageUrl={p.image_url} className="size-full text-lg" />
        <span className="absolute bottom-0 left-0 right-0 bg-region/90 py-1 text-center text-[11px] font-bold uppercase tracking-wider">
          {p.region.name}
        </span>
      </div>

      <div className="flex flex-1 flex-col gap-2 p-3">
        <h3 className="line-clamp-2 min-h-10 text-sm font-semibold leading-5" title={p.name}>
          {p.name}
        </h3>
        <p className="text-xs text-muted">{p.platform.name}</p>

        <div className="mt-auto">
          {p.discount_percent > 0 && p.msrp_cents && (
            <p className="flex items-center gap-2 text-xs">
              <span className="text-muted line-through">{formatPrice(p.msrp_cents)}</span>
              <span className="rounded bg-discount px-1.5 py-0.5 font-bold">-{p.discount_percent}%</span>
            </p>
          )}
          <p className="mt-1 text-xs text-muted">
            From <span className="text-xl font-bold text-white">{formatPrice(p.price_cents)}</span>
          </p>
          <p className="text-[11px] text-muted">
            {p.offer_count} {p.offer_count === 1 ? "offer" : "offers"}
          </p>
        </div>
      </div>
    </Link>
  );
}
