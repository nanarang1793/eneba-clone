import { useEffect } from "react";
import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { getOffers, getProduct } from "../../api/catalog";
import { Cover } from "../../components/ui/Cover";
import { formatPrice } from "../../components/ui/Price";
import { OffersTable } from "./OffersTable";

const TYPE_LABELS = { game: "Game", dlc: "DLC", gift_card: "Gift card", subscription: "Subscription" } as const;

/**
 * /product/:slug: one product, its price box, other platforms/regions of the
 * same game, and every seller's offer. Product and offers are two separate
 * requests, fetched in parallel, so each can load or fail on its own.
 */
export function ProductPage() {
  const { slug = "" } = useParams();
  const product = useQuery({ queryKey: ["product", slug], queryFn: () => getProduct(slug) });
  const offers = useQuery({ queryKey: ["offers", slug], queryFn: () => getOffers(slug) });

  const p = product.data;

  // Browser tab title, e.g. "Elden Ring (PC) Steam Key GLOBAL | Eneba clone".
  useEffect(() => {
    if (p) document.title = `${p.name} | Eneba clone`;
    return () => {
      document.title = "Eneba clone";
    };
  }, [p]);

  if (product.isError) {
    const notFound = product.error instanceof ApiError && product.error.status === 404;
    return (
      <div className="py-24 text-center">
        <h1 className="text-2xl font-bold">{notFound ? "Product not found" : "Couldn't load this product"}</h1>
        <p className="mt-2 text-muted">{notFound ? "It may have been removed." : product.error.message}</p>
        <Link to="/store" className="mt-6 inline-block rounded-md bg-cta px-5 py-2 font-semibold text-page">
          Back to the store
        </Link>
      </div>
    );
  }

  if (!p) {
    return (
      <div className="grid animate-pulse gap-8 md:grid-cols-[280px_1fr]">
        <div className="aspect-[3/4] rounded-lg bg-surface" />
        <div className="space-y-4">
          <div className="h-8 w-2/3 rounded bg-surface" />
          <div className="h-5 w-1/3 rounded bg-surface" />
          <div className="h-40 rounded-lg bg-surface" />
        </div>
      </div>
    );
  }

  return (
    <article className="space-y-10">
      {/* Breadcrumb: Store › Steam › Elden Ring */}
      <nav aria-label="Breadcrumb" className="text-sm text-muted">
        <Link to="/store" className="hover:text-white">Store</Link>
        <span className="mx-2">›</span>
        <Link to={`/store?platform=${p.platform.slug}`} className="hover:text-white">{p.platform.name}</Link>
        <span className="mx-2">›</span>
        <span className="text-white">{p.title}</span>
      </nav>

      <div className="grid gap-8 md:grid-cols-[280px_1fr] lg:grid-cols-[280px_1fr_320px]">
        <Cover title={p.title} imageUrl={p.image_url} className="aspect-[3/4] w-full max-w-[280px] rounded-lg text-2xl" />

        {/* Name, tags, facts, other editions */}
        <div className="space-y-5">
          <h1 className="text-3xl font-bold leading-tight">{p.name}</h1>

          <ul className="flex flex-wrap gap-2 text-xs font-semibold">
            <li className="rounded bg-surface px-2.5 py-1">{p.platform.name}</li>
            <li className="rounded bg-region/20 px-2.5 py-1 text-region">Works in: {p.region.name}</li>
            <li className="rounded bg-surface px-2.5 py-1">{TYPE_LABELS[p.type]}</li>
          </ul>

          <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
            {p.publisher && (
              <>
                <dt className="text-muted">Publisher</dt>
                <dd>{p.publisher}</dd>
              </>
            )}
            {p.release_date && (
              <>
                <dt className="text-muted">Release date</dt>
                <dd>{new Date(p.release_date).toLocaleDateString("en-GB", { dateStyle: "long", timeZone: "UTC" })}</dd>
              </>
            )}
          </dl>

          {p.variants.length > 0 && (
            <div>
              <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-muted">Other platforms & regions</h2>
              <ul className="flex flex-wrap gap-2">
                {p.variants.map((v) => (
                  <li key={v.slug}>
                    <Link
                      to={`/product/${v.slug}`}
                      className="block rounded-md border border-line px-3 py-1.5 text-sm hover:border-accent hover:bg-surface"
                    >
                      {v.platform} · {v.region} <span className="font-semibold">{formatPrice(v.price_cents)}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        {/* Price box */}
        <aside className="h-fit rounded-lg bg-surface p-5 md:col-span-2 lg:col-span-1">
          {p.in_stock && p.price_cents !== null ? (
            <>
              {p.discount_percent > 0 && p.msrp_cents && (
                <p className="flex items-center gap-2 text-sm">
                  <span className="text-muted line-through">{formatPrice(p.msrp_cents)}</span>
                  <span className="rounded bg-discount px-1.5 py-0.5 text-xs font-bold">-{p.discount_percent}%</span>
                </p>
              )}
              <p className="mt-1 text-sm text-muted">
                From <span className="text-3xl font-bold text-white">{formatPrice(p.price_cents)}</span>
              </p>
              <p className="mt-1 text-sm text-muted">
                {p.offer_count} {p.offer_count === 1 ? "seller" : "sellers"} in stock
              </p>
              <a
                href="#offers"
                className="mt-4 block rounded-md bg-cta py-3 text-center font-semibold text-page hover:bg-cta-hover"
              >
                Compare offers
              </a>
            </>
          ) : (
            <>
              <p className="text-xl font-bold">Out of stock</p>
              <p className="mt-1 text-sm text-muted">No seller has a key for this product right now.</p>
            </>
          )}
        </aside>
      </div>

      <section id="offers" className="scroll-mt-24">
        <h2 className="mb-4 text-xl font-bold">
          Offers{offers.data && <span className="ml-2 text-base font-normal text-muted">({offers.data.length})</span>}
        </h2>
        {offers.isError ? (
          <p className="rounded-lg bg-surface p-6 text-center text-muted">Couldn't load offers: {offers.error.message}</p>
        ) : !offers.data ? (
          <div className="h-40 animate-pulse rounded-lg bg-surface" />
        ) : (
          <OffersTable offers={offers.data} product={p} />
        )}
      </section>

      <section>
        <h2 className="mb-3 text-xl font-bold">About this product</h2>
        <p className="max-w-3xl leading-7 text-muted">{p.description}</p>
      </section>
    </article>
  );
}
