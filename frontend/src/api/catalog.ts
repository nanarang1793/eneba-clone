import { apiGet } from "./client";
import type { Filters, Offer, Product, ProductDetail, ProductList } from "./types";

export const getProducts = (params: URLSearchParams) => apiGet<ProductList>("/products", params);

export const getFilters = () => apiGet<Filters>("/filters");

// encodeURIComponent: a slug comes from the URL, so never paste it in raw.
export const getProduct = (slug: string) => apiGet<ProductDetail>(`/products/${encodeURIComponent(slug)}`);

export const getOffers = (slug: string) =>
  apiGet<{ items: Offer[] }>(`/products/${encodeURIComponent(slug)}/offers`).then((r) => r.items);

export const getSuggestions = (q: string) =>
  apiGet<{ items: Product[] }>("/search/suggest", new URLSearchParams({ q, limit: "6" })).then((r) => r.items);
