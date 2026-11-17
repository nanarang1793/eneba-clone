import { apiGet } from "./client";
import type { Filters, ProductList } from "./types";

export const getProducts = (params: URLSearchParams) => apiGet<ProductList>("/products", params);

export const getFilters = () => apiGet<Filters>("/filters");
