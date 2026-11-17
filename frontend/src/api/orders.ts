import { apiGet, apiPost, apiRequest } from "./client";
import type { CartLineInput, Order, Quote } from "./types";

export const quoteCart = (items: CartLineInput[]) => apiPost<Quote>("/cart/quote", { items });

/**
 * The same idempotencyKey for the same checkout attempt means a double click
 * or a retry after a network error returns the first order instead of
 * reserving keys twice.
 */
export const createOrder = (items: CartLineInput[], idempotencyKey: string) =>
  apiRequest<Order>("POST", "/orders", { body: { items }, headers: { "Idempotency-Key": idempotencyKey } });

export const payOrder = (id: string, cardNumber: string) =>
  apiPost<Order>(`/orders/${encodeURIComponent(id)}/pay`, { card_number: cardNumber });

export const getOrder = (id: string) => apiGet<Order>(`/orders/${encodeURIComponent(id)}`);

export const listOrders = () => apiGet<{ items: Order[] }>("/orders").then((r) => r.items);
