import { useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { getFilters, getProducts } from "../../api/catalog";
import { FilterSidebar } from "./FilterSidebar";
import { Pagination } from "./Pagination";
import { ProductCard } from "./ProductCard";
import { SortSelect } from "./SortSelect";
import { PER_PAGE, useStoreParams } from "./useStoreParams";

/**
 * /store: search results + filters.
 * Data flow: URL -> useStoreParams -> query key -> GET /api/products -> render.
 * Changing a filter only changes the URL; the new URL produces a new query key,
 * and TanStack Query fetches (or reuses a cached) result for it.
 */
export function StorePage() {
  const store = useStoreParams();
  const { filters } = store;

  const products = useQuery({
    queryKey: ["products", store.apiParams.toString()],
    queryFn: () => getProducts(store.apiParams),
    // Keep showing the old results while the next page/filter loads,
    // instead of flashing an empty grid.
    placeholderData: keepPreviousData,
  });

  // Platforms, regions and price bounds rarely change: fetch once per visit.
  const meta = useQuery({ queryKey: ["filters"], queryFn: getFilters, staleTime: Infinity });

  const data = products.data;

  // On phones the sidebar is hidden behind a "Filters" button; on desktop (lg) it's always shown.
  const [showFilters, setShowFilters] = useState(false);
  const activeFilterCount =
    filters.platforms.length + filters.regions.length + (filters.minPrice || filters.maxPrice ? 1 : 0);

  function goToPage(page: number) {
    store.setPage(page);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[260px_1fr]">
      <div className={showFilters ? "" : "hidden lg:block"}>
        <FilterSidebar
          meta={meta.data}
          facets={data?.facets}
          selected={filters}
          onToggle={store.toggle}
          onPrice={store.setPriceRange}
          onClear={store.clearFilters}
        />
      </div>

      <section>
        <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold">{filters.q ? `Results for “${filters.q}”` : "Store"}</h1>
            <p className="text-sm text-muted">
              {data ? `${data.total} ${data.total === 1 ? "result" : "results"}` : "Loading…"}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowFilters((v) => !v)}
              aria-expanded={showFilters}
              className="h-9 rounded-md border border-line bg-surface px-3 text-sm lg:hidden"
            >
              Filters{activeFilterCount > 0 && ` (${activeFilterCount})`}
            </button>
            <SortSelect value={filters.sort} onChange={(v) => store.setValue("sort", v)} />
          </div>
        </div>

        {products.isError ? (
          <ErrorBox message={products.error.message} onRetry={() => products.refetch()} />
        ) : !data ? (
          <Grid>
            {Array.from({ length: 12 }, (_, i) => (
              <div key={i} className="aspect-[3/5] animate-pulse rounded-lg bg-surface" />
            ))}
          </Grid>
        ) : data.items.length === 0 ? (
          <EmptyState onClear={store.clearFilters} searched={filters.q !== ""} />
        ) : (
          <>
            {/* Dim the old results while the new ones load. */}
            <div
              aria-busy={products.isPlaceholderData}
              className={products.isPlaceholderData ? "opacity-60 transition-opacity" : "transition-opacity"}
            >
              <Grid>
                {data.items.map((p) => (
                  <ProductCard key={p.id} product={p} />
                ))}
              </Grid>
            </div>
            <Pagination page={filters.page} totalPages={data.total_pages} onChange={goToPage} />
            <p className="mt-3 text-center text-xs text-muted">
              Showing {(filters.page - 1) * PER_PAGE + 1}–{(filters.page - 1) * PER_PAGE + data.items.length} of{" "}
              {data.total}
            </p>
          </>
        )}
      </section>
    </div>
  );
}

function Grid({ children }: { children: React.ReactNode }) {
  return <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4">{children}</div>;
}

function EmptyState({ onClear, searched }: { onClear: () => void; searched: boolean }) {
  return (
    <div className="rounded-lg bg-surface px-6 py-16 text-center">
      <p className="text-lg font-semibold">No products match</p>
      <p className="mt-1 text-sm text-muted">
        {searched ? "Try another spelling or remove some filters." : "Try removing some filters."}
      </p>
      <button
        onClick={onClear}
        className="mt-5 rounded-md bg-cta px-5 py-2 text-sm font-semibold text-page hover:bg-cta-hover"
      >
        Clear filters
      </button>
    </div>
  );
}

function ErrorBox({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div role="alert" className="rounded-lg border border-discount/60 bg-discount/10 px-6 py-10 text-center">
      <p className="font-semibold">Couldn't load products</p>
      <p className="mt-1 text-sm text-muted">{message}</p>
      <button onClick={onRetry} className="mt-4 rounded-md bg-surface px-4 py-2 text-sm hover:bg-surface-hover">
        Try again
      </button>
    </div>
  );
}
