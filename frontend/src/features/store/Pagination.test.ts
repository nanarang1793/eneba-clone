import { describe, expect, it } from "vitest";
import { pageList } from "./Pagination";

describe("pageList", () => {
  it("shows every page when there are few", () => {
    expect(pageList(1, 3)).toEqual([1, 2, 3]);
  });

  it("always shows first and last, the neighbours of the current page, and … for gaps", () => {
    expect(pageList(5, 9)).toEqual([1, "…", 4, 5, 6, "…", 9]);
    expect(pageList(1, 9)).toEqual([1, 2, "…", 9]);
    expect(pageList(9, 9)).toEqual([1, "…", 8, 9]);
  });
});
