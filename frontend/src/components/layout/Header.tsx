import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";

/**
 * Top bar: logo, search, account/cart placeholders.
 * Submitting the search goes to /store?q=… (a fresh search drops old filters).
 */
export function Header() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const urlQuery = searchParams.get("q") ?? "";
  const [text, setText] = useState(urlQuery);

  // Keep the box in sync when the URL changes (back button, "clear search"…).
  useEffect(() => setText(urlQuery), [urlQuery]);

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    const q = text.trim();
    navigate(q ? `/store?q=${encodeURIComponent(q)}` : "/store");
  }

  return (
    <header className="sticky top-0 z-20 border-b border-line bg-header/95 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center gap-4 px-4">
        <Link to="/" className="flex shrink-0 items-baseline gap-1.5 text-2xl font-extrabold tracking-tight">
          eneba
          <span className="rounded bg-accent/20 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-accent">
            clone
          </span>
        </Link>

        <form onSubmit={onSubmit} role="search" className="relative mx-auto w-full max-w-2xl">
          <input
            type="search"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Search for games, gift cards and more"
            aria-label="Search products"
            className="h-11 w-full rounded-lg border border-line bg-page pl-11 pr-24 text-sm text-white placeholder:text-muted focus:border-accent focus:outline-none"
          />
          <SearchIcon className="pointer-events-none absolute left-3.5 top-1/2 size-5 -translate-y-1/2 text-muted" />
          <button
            type="submit"
            className="absolute right-1.5 top-1/2 h-8 -translate-y-1/2 rounded-md bg-cta px-4 text-sm font-semibold text-page hover:bg-cta-hover"
          >
            Search
          </button>
        </form>

        {/* Wired up on day 4 (auth) and day 5 (cart). */}
        <nav className="hidden shrink-0 items-center gap-5 text-sm text-muted md:flex">
          <Link to="/store" className="hover:text-white">
            Store
          </Link>
          <span className="cursor-not-allowed opacity-60" title="Coming soon">
            Cart
          </span>
          <span className="cursor-not-allowed opacity-60" title="Coming soon">
            Sign in
          </span>
        </nav>
      </div>
    </header>
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
