import { beforeEach, describe, expect, it } from "vitest";
import { MAX_QTY, useCart } from "./cartStore";

const line = { offer_id: 7, slug: "elden-ring", name: "Elden Ring", title: "Elden Ring", seller_name: "KeyHaven", price_cents: 3169 };

describe("cart store", () => {
  beforeEach(() => useCart.getState().clear());

  it("adds a line, and adding the same offer again increases its quantity", () => {
    useCart.getState().add(line);
    useCart.getState().add(line);
    expect(useCart.getState().lines).toHaveLength(1);
    expect(useCart.getState().lines[0].qty).toBe(2);
  });

  it(`never goes above ${MAX_QTY} or below 1`, () => {
    useCart.getState().add(line);
    useCart.getState().setQty(7, 99);
    expect(useCart.getState().lines[0].qty).toBe(MAX_QTY);
    useCart.getState().setQty(7, 0);
    expect(useCart.getState().lines[0].qty).toBe(1);
  });

  it("removes a line", () => {
    useCart.getState().add(line);
    useCart.getState().add({ ...line, offer_id: 8 });
    useCart.getState().remove(7);
    expect(useCart.getState().lines.map((l) => l.offer_id)).toEqual([8]);
  });
});
