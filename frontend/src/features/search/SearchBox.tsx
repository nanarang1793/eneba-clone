import { useEffect, useState, type FormEvent, type KeyboardEvent } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { getSuggestions } from "../../api/catalog";
import { Cover } from "../../components/ui/Cover";
import { formatPrice } from "../../components/ui/Price";
import { useDebouncedValue } from "./useDebouncedValue";

const MIN_CHARS = 2; // same rule as the backend

/**
 * Header search with a suggestions dropdown.
 *
 * - Typing waits 250 ms (debounce), then asks GET /api/search/suggest.
 * - ↑/↓ move through the suggestions, Enter opens the highlighted product,
 *   Escape closes the list. Enter with nothing highlighted, or the last row,
 *   goes to the full results page /store?q=…
 * - Follows the ARIA "combobox" pattern so screen readers announce it.
 */
export function SearchBox() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const urlQuery = searchParams.get("q") ?? "";

  const [text, setText] = useState(urlQuery);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1); // highlighted row, -1 = none

  // Keep the box in sync when the URL changes (back button, "clear search"…).
  useEffect(() => setText(urlQuery), [urlQuery]);

  const query = text.trim();
  const debounced = useDebouncedValue(query, 250);
  const suggestions = useQuery({
    queryKey: ["suggest", debounced],
    queryFn: () => getSuggestions(debounced),
    enabled: debounced.length >= MIN_CHARS,
    placeholderData: keepPreviousData, // keep the old list while the next one loads
    staleTime: 60_000,
  });

  const items = query.length >= MIN_CHARS ? (suggestions.data ?? []) : [];
  const showList = open && query.length >= MIN_CHARS;
  const rowCount = items.length + 1; // the suggestions + "See all results"

  function close() {
    setOpen(false);
    setActive(-1);
  }

  function goToResults() {
    close();
    navigate(query ? `/store?q=${encodeURIComponent(query)}` : "/store");
  }

  function goToProduct(slug: string) {
    close();
    navigate(`/product/${slug}`);
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (active >= 0 && active < items.length) goToProduct(items[active].slug);
    else goToResults();
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      if (!showList) return;
      e.preventDefault(); // don't move the text cursor
      const step = e.key === "ArrowDown" ? 1 : -1;
      setActive((i) => (i + step + rowCount) % rowCount); // wrap around
    } else if (e.key === "Escape" && showList) {
      // Browsers also clear a search box on Escape. The first Escape should
      // only close the list; a second one clears the text as usual.
      e.preventDefault();
      close();
    }
  }

  return (
    <form onSubmit={onSubmit} role="search" className="relative mx-auto w-full max-w-2xl">
      <input
        type="search"
        value={text}
        onChange={(e) => {
          setText(e.target.value);
          setOpen(true);
          setActive(-1);
        }}
        onFocus={() => setOpen(true)}
        onBlur={close}
        onKeyDown={onKeyDown}
        placeholder="Search for games, gift cards and more"
        aria-label="Search products"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={showList}
        aria-controls="search-suggestions"
        aria-activedescendant={active >= 0 ? `suggestion-${active}` : undefined}
        autoComplete="off"
        className="h-11 w-full rounded-lg border border-line bg-page pl-11 pr-24 text-sm text-white placeholder:text-muted focus:border-accent focus:outline-none"
      />
      <SearchIcon className="pointer-events-none absolute left-3.5 top-1/2 size-5 -translate-y-1/2 text-muted" />
      <button
        type="submit"
        className="absolute right-1.5 top-1/2 h-8 -translate-y-1/2 rounded-md bg-cta px-4 text-sm font-semibold text-page hover:bg-cta-hover"
      >
        Search
      </button>

      {showList && (
        <ul
          id="search-suggestions"
          role="listbox"
          className="absolute left-0 right-0 top-full z-30 mt-2 overflow-hidden rounded-lg border border-line bg-surface shadow-2xl"
          // mousedown fires before the input's blur; preventing it keeps the
          // list open long enough for the click to land.
          onMouseDown={(e) => e.preventDefault()}
        >
          {items.map((p, i) => (
            <li
              key={p.id}
              id={`suggestion-${i}`}
              role="option"
              aria-selected={i === active}
              onClick={() => goToProduct(p.slug)}
              onMouseEnter={() => setActive(i)}
              className={`flex cursor-pointer items-center gap-3 px-3 py-2 ${i === active ? "bg-surface-hover" : ""}`}
            >
              <Cover title={p.title} imageUrl={p.image_url} showTitle={false} className="h-12 w-9 shrink-0 rounded" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{p.name}</p>
                <p className="text-xs text-muted">
                  {p.platform.name} · {p.region.name}
                </p>
              </div>
              <span className="shrink-0 text-sm font-bold">{formatPrice(p.price_cents)}</span>
            </li>
          ))}

          {items.length === 0 && !suggestions.isFetching && (
            <li className="px-3 py-3 text-sm text-muted">No matches for “{query}”</li>
          )}

          <li
            id={`suggestion-${items.length}`}
            role="option"
            aria-selected={active === items.length}
            onClick={goToResults}
            onMouseEnter={() => setActive(items.length)}
            className={`cursor-pointer border-t border-line px-3 py-2.5 text-sm text-accent ${
              active === items.length ? "bg-surface-hover" : ""
            }`}
          >
            See all results for “{query}” →
          </li>
        </ul>
      )}
    </form>
  );
}

function SearchIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className={className} aria-hidden>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" strokeLinecap="round" />
    </svg>
  );
}
