import { Link, useLocation } from "react-router";
import { useCurrentUser } from "../../features/auth/useCurrentUser";
import { useCartCount } from "../../features/cart/cartStore";
import { SearchBox } from "../../features/search/SearchBox";

/** Top bar: logo, search with suggestions, cart placeholder, account link. */
export function Header() {
  const { user, isLoading } = useCurrentUser();
  const cartCount = useCartCount();
  const location = useLocation();
  // "Sign in" brings you back to the page you were on.
  const here = location.pathname + location.search;
  const loginLink = here.startsWith("/login") || here.startsWith("/register") ? "/login" : `/login?next=${encodeURIComponent(here)}`;

  return (
    <header className="sticky top-0 z-20 border-b border-line bg-header/95 backdrop-blur">
      {/* Phones: logo + icons on one row, search full-width below.
          md and up: one row with the search in the middle. */}
      <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-x-4 gap-y-3 px-4 py-3 md:h-16 md:flex-nowrap md:py-0">
        <Link to="/" className="flex shrink-0 items-baseline gap-1.5 text-2xl font-extrabold tracking-tight">
          eneba
          <span className="rounded bg-accent/20 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-accent">
            clone
          </span>
        </Link>

        <div className="order-last w-full md:order-none md:flex-1">
          <SearchBox />
        </div>

        <nav className="ml-auto flex shrink-0 items-center gap-5 text-sm text-muted md:ml-0">
          <Link to="/store" className="hidden hover:text-white md:block">
            Store
          </Link>
          <Link to="/cart" className="relative flex items-center gap-1.5 hover:text-white" aria-label={`Cart, ${cartCount} items`}>
            <CartIcon className="size-6" />
            <span className="hidden md:inline">Cart</span>
            {cartCount > 0 && (
              <span className="absolute -top-2 left-3.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-cta px-1 text-[11px] font-bold text-page">
                {cartCount}
              </span>
            )}
          </Link>
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

function CartIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} className={className} aria-hidden>
      <path d="M3 4h2l2.4 11.2a1 1 0 0 0 1 .8h8.9a1 1 0 0 0 1-.8L20 8H6.2" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="9" cy="19.5" r="1.3" />
      <circle cx="17" cy="19.5" r="1.3" />
    </svg>
  );
}
