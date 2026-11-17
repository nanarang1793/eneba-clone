import { useSearchParams } from "react-router";

/*
 * The URL is the single source of truth for the store's filters.
 *
 *   /store?q=elden&platform=steam&platform=xbox&min_price=10&sort=price_asc&page=2
 *
 * Why: a filtered page can be shared or bookmarked, refresh keeps it, and the
 * browser's back button undoes the last filter change. There is no separate
 * React state that could get out of sync with the address bar.
 *
 * The URL uses the same parameter names as GET /api/products, so the API
 * request is just the URL's query string, minus any unknown keys.
 */

export const PER_PAGE = 24;

// Keys we forward to the API. Anything else in the URL is ignored.
const API_KEYS = ["q", "platform", "region", "min_price", "max_price", "sort", "page"] as const;

export type ListKey = "platform" | "region";

export function useStoreParams() {
  const [searchParams, setSearchParams] = useSearchParams();

  const filters = {
    q: searchParams.get("q") ?? "",
    platforms: searchParams.getAll("platform"),
    regions: searchParams.getAll("region"),
    minPrice: searchParams.get("min_price") ?? "",
    maxPrice: searchParams.get("max_price") ?? "",
    sort: searchParams.get("sort") ?? "relevance",
    page: Math.max(1, Number(searchParams.get("page")) || 1),
  };

  /** The query string for GET /api/products. */
  const apiParams = new URLSearchParams();
  for (const key of API_KEYS) {
    for (const value of searchParams.getAll(key)) apiParams.append(key, value);
  }
  apiParams.set("per_page", String(PER_PAGE));

  /**
   * Apply changes to the URL. Every filter change goes back to page 1,
   * because page 5 of the old results means nothing for the new ones.
   * Empty values are removed so the URL stays short.
   */
  function update(change: (next: URLSearchParams) => void, keepPage = false) {
    const next = new URLSearchParams(searchParams);
    change(next);
    if (!keepPage) next.delete("page");
    for (const [key, value] of [...next.entries()]) {
      if (value === "") next.delete(key, value);
    }
    setSearchParams(next);
  }

  function setValue(key: "q" | "min_price" | "max_price" | "sort", value: string) {
    update((next) => next.set(key, value));
  }

  /** Both price bounds in one URL change (two setValue calls would overwrite each other). */
  function setPriceRange(min: string, max: string) {
    update((next) => {
      next.set("min_price", min);
      next.set("max_price", max);
    });
  }

  /** Tick or untick a checkbox (platform/region can have several values). */
  function toggle(key: ListKey, slug: string) {
    update((next) => {
      if (next.getAll(key).includes(slug)) next.delete(key, slug);
      else next.append(key, slug);
    });
  }

  function setPage(page: number) {
    // Page 1 is the default, so it's left out of the URL.
    update((next) => next.set("page", page > 1 ? String(page) : ""), true);
  }

  function clearFilters() {
    // Keep the search text and sort; drop everything the sidebar controls.
    update((next) => {
      for (const key of ["platform", "region", "min_price", "max_price"]) next.delete(key);
    });
  }

  return { filters, apiParams, setValue, setPriceRange, toggle, setPage, clearFilters };
}
