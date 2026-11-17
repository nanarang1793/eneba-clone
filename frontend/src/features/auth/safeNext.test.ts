import { describe, expect, it } from "vitest";
import { safeNext } from "./useCurrentUser";

// After signing in we redirect to ?next=… — it must never leave our site.
describe("safeNext", () => {
  it("keeps paths on this site", () => {
    expect(safeNext("/cart")).toBe("/cart");
    expect(safeNext("/product/elden-ring?x=1")).toBe("/product/elden-ring?x=1");
  });

  it("falls back to /account for anything else", () => {
    for (const evil of ["//evil.com", "/\\evil.com", "https://evil.com", "javascript:alert(1)", "evil.com", "", null]) {
      expect(safeNext(evil)).toBe("/account");
    }
  });
});
