import { describe, expect, it } from "vitest";
import type { Category } from "../../api/types";
import { localizeCategories } from "./categoryName";

const category = (slug: string, name: string): Category => ({
  id: slug,
  slug,
  name,
  created_at: "2026-06-30T00:00:00Z",
  updated_at: "2026-06-30T00:00:00Z",
});

describe("localizeCategories", () => {
  it("localizes defaults by slug and preserves custom names", () => {
    const categories = [
      category("food", "Canonical food"),
      category("home-repair", "Home repair"),
    ];

    expect(localizeCategories(categories, "ru").map(({ name }) => name)).toEqual([
      "Еда",
      "Home repair",
    ]);
    expect(localizeCategories(categories, "en").map(({ name }) => name)).toEqual([
      "Food",
      "Home repair",
    ]);
  });
});
