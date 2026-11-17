import { create } from "zustand";
import { persist } from "zustand/middleware";

/**
 * The cart lives in the browser (localStorage), so guests can fill it
 * without signing in and it survives a refresh.
 *
 * Each line remembers what the product looked like when it was added
 * (name, seller, price) only to draw the cart instantly. The server never
 * trusts these: the cart page asks POST /api/cart/quote for real prices and
 * stock, and the order is priced from the database again.
 */
export type CartLine = {
  offer_id: number;
  qty: number;
  slug: string;
  name: string;
  title: string;
  seller_name: string;
  price_cents: number; // price when added, to show "price changed"
};

export const MAX_QTY = 10; // same limit as the backend

type CartState = {
  lines: CartLine[];
  add: (line: Omit<CartLine, "qty">) => void;
  setQty: (offerId: number, qty: number) => void;
  remove: (offerId: number) => void;
  clear: () => void;
};

export const useCart = create<CartState>()(
  persist(
    (set) => ({
      lines: [],
      // Adding an offer that's already in the cart adds one more of it.
      add: (line) =>
        set((s) => {
          const existing = s.lines.find((l) => l.offer_id === line.offer_id);
          if (existing) {
            return {
              lines: s.lines.map((l) =>
                l.offer_id === line.offer_id ? { ...l, qty: Math.min(l.qty + 1, MAX_QTY) } : l,
              ),
            };
          }
          return { lines: [...s.lines, { ...line, qty: 1 }] };
        }),
      setQty: (offerId, qty) =>
        set((s) => ({
          lines: s.lines.map((l) =>
            l.offer_id === offerId ? { ...l, qty: Math.max(1, Math.min(qty, MAX_QTY)) } : l,
          ),
        })),
      remove: (offerId) => set((s) => ({ lines: s.lines.filter((l) => l.offer_id !== offerId) })),
      clear: () => set({ lines: [] }),
    }),
    { name: "eneba-cart" }, // the localStorage key
  ),
);

/** Total number of keys in the cart, for the header badge. */
export const useCartCount = () => useCart((s) => s.lines.reduce((n, l) => n + l.qty, 0));
