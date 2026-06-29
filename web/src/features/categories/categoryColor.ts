const categoryColorCount = 8;

export function categoryColorClass(categoryKey: string) {
  return `category-color-${categoryColorIndex(categoryKey)}`;
}

export function categoryColorIndex(categoryKey: string) {
  let hash = 2166136261;
  for (let index = 0; index < categoryKey.length; index += 1) {
    hash ^= categoryKey.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0) % categoryColorCount;
}
