import { useEffect } from "react";
import { useLocation } from "react-router";

/**
 * Single-page apps keep the scroll position when the route changes, so
 * opening a product from the bottom of the store landed mid-page. Scroll to
 * the top whenever the path changes. Only the path, not ?query changes:
 * ticking a filter shouldn't jump the page.
 */
export function ScrollToTop() {
  const { pathname } = useLocation();
  useEffect(() => {
    // Braces matter: newer browsers make scrollTo() return a Promise, and an
    // effect that returns anything but a cleanup function crashes React.
    window.scrollTo(0, 0);
  }, [pathname]);
  return null;
}
