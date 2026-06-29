import { categoryColorClass } from "../../categories/categoryColor";

export function CategoryBadge({
  categoryKey,
  name,
}: {
  categoryKey: string;
  name: string;
}) {
  return (
    <span
      className={`tag category-badge ${categoryColorClass(categoryKey)}`}
    >
      {name}
    </span>
  );
}
