import type { Product } from "../../api/types";
import { formatPrice } from "../../components/ui/Price";

/** One product tile in the grid, laid out like Eneba's cards. */
export function ProductCard({ product }: { product: Product }) {
  const p = product;
  return (
    <article className="group flex flex-col overflow-hidden rounded-lg bg-surface transition hover:-translate-y-0.5 hover:bg-surface-hover">
      <div className="relative aspect-[3/4]">
        {p.image_url ? (
          <img src={p.image_url} alt="" loading="lazy" className="size-full object-cover" />
        ) : (
          <PlaceholderCover product={p} />
        )}
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
    </article>
  );
}

/**
 * The seed has no cover art (we don't copy Eneba's images), so each product
 * gets a gradient whose colour comes from its title: every variant of
 * "Elden Ring" gets the same colour.
 */
function PlaceholderCover({ product }: { product: Product }) {
  let hash = 0;
  for (const ch of product.title) hash = (hash * 31 + ch.charCodeAt(0)) % 360;
  return (
    <div
      className="flex size-full items-center justify-center p-4 text-center"
      style={{ background: `linear-gradient(160deg, hsl(${hash} 60% 45%), hsl(${(hash + 60) % 360} 55% 22%))` }}
    >
      <span className="text-lg font-extrabold leading-tight drop-shadow-md">{product.title}</span>
    </div>
  );
}
