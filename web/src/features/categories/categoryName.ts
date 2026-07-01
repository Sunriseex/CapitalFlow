import type { Category } from "../../api/types";
import { getDictionary, type Locale } from "../../shared/i18n/i18n";

export function localizeCategories(
  categories: Category[],
  locale: Locale,
): Category[] {
  const names = getDictionary(locale).defaultCategories;
  return categories.map((category) => ({
    ...category,
    name: names[category.slug as keyof typeof names] ?? category.name,
  }));
}
