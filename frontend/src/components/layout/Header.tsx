import { Link } from "react-router";
import { SearchBox } from "../../features/search/SearchBox";

/** Top bar: logo, search with suggestions, account/cart placeholders. */
export function Header() {
  return (
    <header className="sticky top-0 z-20 border-b border-line bg-header/95 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center gap-4 px-4">
        <Link to="/" className="flex shrink-0 items-baseline gap-1.5 text-2xl font-extrabold tracking-tight">
          eneba
          <span className="rounded bg-accent/20 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-accent">
            clone
          </span>
        </Link>

        <SearchBox />

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
