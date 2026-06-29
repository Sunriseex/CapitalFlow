import { describe, expect, it } from "vitest";
import { categoryColorClass, categoryColorIndex } from "./categoryColor";

describe("category colors", () => {
  it("assigns a stable palette color", () => {
    expect(categoryColorClass("category-food")).toBe(
      categoryColorClass("category-food"),
    );
    expect(categoryColorIndex("category-food")).toBeGreaterThanOrEqual(0);
    expect(categoryColorIndex("category-food")).toBeLessThan(8);
  });

  it("distributes category keys across the palette", () => {
    const colors = new Set(
      [
        "food",
        "transport",
        "housing",
        "health",
        "education",
        "subscriptions",
        "salary",
        "entertainment",
      ].map(categoryColorIndex),
    );
    expect(colors.size).toBeGreaterThanOrEqual(5);
  });
});
