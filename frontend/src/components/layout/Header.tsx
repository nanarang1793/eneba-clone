import { Link, useLocation } from "react-router";
import { useCurrentUser } from "../../features/auth/useCurrentUser";
import { SearchBox } from "../../features/search/SearchBox";

/** Top bar: logo, search with suggestions, cart placeholder, account link. */
export function Header() {
  const { user, isLoading } = useCurrentUser();
  const location = useLocation();
  // "Sign in" brings you back to the page you were on.
  const here = location.pathname + location.search;
  const loginLink = here.startsWith("/login") || here.startsWith("/register") ? "/login" : `/login?next=${encodeURIComponent(here)}`;

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

        <nav className="flex shrink-0 items-center gap-5 text-sm text-muted">
          <Link to="/store" className="hidden hover:text-white md:block">
            Store
          </Link>
          {/* The cart arrives on day 5. */}
          <span className="hidden cursor-not-allowed opacity-60 md:block" title="Coming soon">
            Cart
          </span>
          {isLoading ? (
            <span className="w-16" /> // keep the layout still while we check the session
          ) : user ? (
            <Link
              to="/account"
              className="flex items-center gap-2 font-semibold text-white hover:text-accent"
              title="My account"
            >
              <span className="flex size-8 items-center justify-center rounded-full bg-accent text-sm font-bold uppercase text-page">
                {user.display_name.charAt(0)}
              </span>
              <span className="hidden max-w-32 truncate lg:block">{user.display_name}</span>
            </Link>
          ) : (
            <Link to={loginLink} className="font-semibold text-white hover:text-accent">
              Sign in
            </Link>
          )}
        </nav>
      </div>
    </header>
  );
}
