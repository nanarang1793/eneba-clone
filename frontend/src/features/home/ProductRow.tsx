import { Link } from "react-router";
import type { Product } from "../../api/types";
import { ProductCard } from "../store/ProductCard";

/** A titled, horizontally scrolling row of product cards with "See all". */
export function ProductRow({ title, products, seeAllUrl }: { title: string; products: Product[]; seeAllUrl: string }) {
  if (products.length === 0) return null; // nothing to show: hide the whole row

  return (
    <section aria-label={title}>
      <div className="mb-4 flex items-end justify-between">
        <h2 className="text-2xl font-bold">{title}</h2>
        <Link to={seeAllUrl} className="text-sm font-semibold text-accent hover:underline">
          See all ›
        </Link>
      </div>
      {/* Scroll sideways; each card snaps into place on touch screens. */}
      <ul className="-mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-2 [scrollbar-width:thin]">
        {products.map((p) => (
          <li key={p.id} className="w-40 shrink-0 snap-start sm:w-48">
            <ProductCard product={p} />
          </li>
        ))}
      </ul>
    </section>
  );
}
