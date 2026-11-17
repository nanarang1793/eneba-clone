import { Link, Outlet, useLocation } from "react-router";
import { ErrorBoundary } from "./ErrorBoundary";
import { Header } from "./Header";
import { ScrollToTop } from "./ScrollToTop";

/** Shared page frame: header, the current route's page in <Outlet />, footer. */
export function Layout() {
  const { pathname } = useLocation();
  return (
    <div className="flex min-h-screen flex-col">
      <ScrollToTop />
      <Header />
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6">
        {/* key: moving to another page clears a previous page's crash */}
        <ErrorBoundary key={pathname}>
          <Outlet />
        </ErrorBoundary>
      </main>
      <footer className="border-t border-line">
        <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-8 text-sm text-muted md:flex-row md:items-center md:justify-between">
          <nav className="flex flex-wrap gap-x-6 gap-y-2">
            <Link to="/store" className="hover:text-white">Store</Link>
            <Link to="/store?type=gift_card" className="hover:text-white">Gift cards</Link>
            <Link to="/store?type=subscription" className="hover:text-white">Subscriptions</Link>
            <Link to="/account" className="hover:text-white">My account</Link>
          </nav>
          <p className="text-xs">
            Eneba clone, built for the Eneba Intern Assignment 2027. Not affiliated with Eneba. Prices and keys are fake.
          </p>
        </div>
      </footer>
    </div>
  );
}
