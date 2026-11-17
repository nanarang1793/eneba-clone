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
