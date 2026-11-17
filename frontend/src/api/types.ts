// Mirrors backend/internal/catalog/types.go. Prices are integer cents.

export type Named = { slug: string; name: string };

export type Product = {
  id: number;
  slug: string;
  name: string; // "Elden Ring (PC) Steam Key GLOBAL"
  title: string; // "Elden Ring"
  type: "game" | "dlc" | "gift_card" | "subscription";
  image_url: string | null;
  platform: Named;
  region: Named;
  price_cents: number; // cheapest in-stock offer
  msrp_cents: number | null;
  discount_percent: number;
  offer_count: number;
};

export type FacetCount = Named & { count: number };

export type ProductList = {
  items: Product[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
  facets: { platforms: FacetCount[]; regions: FacetCount[]; types: FacetCount[] };
};

export type Filters = {
  platforms: Named[];
  regions: Named[];
  types: Named[];
  price: { min_cents: number; max_cents: number };
  sorts: string[];
};

/** GET /api/products/{slug} */
export type ProductDetail = {
  id: number;
  slug: string;
  name: string;
  title: string;
  type: Product["type"];
  description: string;
  image_url: string | null;
  publisher: string | null;
  release_date: string | null; // "2022-02-25"
  platform: Named;
  region: Named;
  msrp_cents: number | null;
  price_cents: number | null; // null when no seller has stock
  discount_percent: number;
  offer_count: number;
  in_stock: boolean;
  variants: { slug: string; platform: string; region: string; price_cents: number }[];
};

/** One row of GET /api/products/{slug}/offers */
export type Offer = {
  id: number;
  price_cents: number;
  stock: number;
  in_stock: boolean;
  is_best: boolean; // cheapest in-stock offer
  seller: { slug: string; name: string; rating: number; sales_count: number };
};

/** A signed-in account (GET /api/auth/me). */
export type User = {
  id: string;
  email: string;
  display_name: string;
  created_at: string;
};

/** Mirrors backend/internal/orders/types.go. */
export type CartLineInput = { offer_id: number; qty: number };

export type ProductRef = {
  slug: string;
  name: string;
  title: string;
  image_url: string | null;
  platform: string;
  region: string;
};

export type QuoteLine = {
  offer_id: number;
  qty: number;
  status: "ok" | "not_enough_stock" | "sold_out" | "unavailable";
  product: ProductRef | null;
  seller_name: string;
  price_cents: number;
  stock: number;
  line_total_cents: number;
};

export type Quote = { items: QuoteLine[]; total_cents: number; all_available: boolean };

export type OrderStatus = "pending" | "paid" | "failed" | "expired";

export type Order = {
  id: string;
  status: OrderStatus;
  total_cents: number;
  currency: string;
  created_at: string;
  expires_at: string;
  paid_at: string | null;
  payment_ref: string | null;
  items: {
    id: number;
    offer_id: number;
    product: ProductRef;
    seller_name: string;
    unit_price_cents: number;
    quantity: number;
    keys: string[];
  }[];
};

/** Homepage (GET /api/home/layout and /api/home/blocks/{id}). */
export type BlockInfo = { id: number; type: string; title: string };

export type Banner = {
  id: number;
  title: string;
  subtitle: string | null;
  image_url: string | null;
  cta_label: string;
  cta_url: string;
};

/** A block's data depends on its type. */
export type Block =
  | (BlockInfo & { type: "banner"; data: { items: Banner[] } })
  | (BlockInfo & { type: "best_sellers" | "deals" | "new_releases" | "by_type"; data: { items: Product[]; see_all_url: string } });
