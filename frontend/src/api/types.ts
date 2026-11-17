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
  facets: { platforms: FacetCount[]; regions: FacetCount[] };
};

export type Filters = {
  platforms: Named[];
  regions: Named[];
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
