import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Search } from "lucide-react";
import type { Category } from "../../api/types";
import { api } from "../../api/client";
import { errorMessage, apiErrorMessages } from "../../shared/api/query";
import { useI18n } from "../../shared/i18n/useI18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";

export function CategoryManager({ categories }: { categories: Category[] }) {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const errors = apiErrorMessages(t);
  const createCategory = useMutation({
    mutationFn: api.createCategory,
    onSuccess: async () => {
      setName("");
      setSlug("");
      setSlugEdited(false);
      await queryClient.invalidateQueries({ queryKey: ["categories"] });
    },
  });
  const visibleCategories = useMemo(() => {
    const normalized = query.trim().toLocaleLowerCase();
    if (!normalized) return categories;
    return categories.filter((category) =>
      `${category.name} ${category.slug}`
        .toLocaleLowerCase()
        .includes(normalized),
    );
  }, [categories, query]);

  return (
    <div className="category-manager">
      <form
        className="category-create-form"
        onSubmit={(event) => {
          event.preventDefault();
          if (!name.trim() || !slug.trim()) return;
          createCategory.mutate({ name: name.trim(), slug: slug.trim() });
        }}
      >
        <div className="field">
          <label htmlFor="category-name">{t.categoriesManagement.name}</label>
          <Input
            id="category-name"
            value={name}
            maxLength={80}
            autoComplete="off"
            placeholder={t.categoriesManagement.namePlaceholder}
            onChange={(event) => {
              const nextName = event.target.value;
              setName(nextName);
              if (!slugEdited) setSlug(categorySlug(nextName));
            }}
          />
        </div>
        <div className="field">
          <label htmlFor="category-slug">{t.categoriesManagement.slug}</label>
          <Input
            id="category-slug"
            value={slug}
            maxLength={80}
            autoComplete="off"
            pattern="[a-z0-9]+(?:[-_][a-z0-9]+)*"
            placeholder="home-repair"
            onChange={(event) => {
              setSlugEdited(true);
              setSlug(event.target.value.toLocaleLowerCase());
            }}
          />
        </div>
        {createCategory.error ? (
          <p className="error inline-error" role="alert">
            {errorMessage(createCategory.error, errors)}
          </p>
        ) : null}
        <Button
          type="submit"
          disabled={!name.trim() || !slug.trim() || createCategory.isPending}
        >
          <Plus aria-hidden="true" />
          {createCategory.isPending
            ? t.categoriesManagement.creating
            : t.categoriesManagement.create}
        </Button>
      </form>

      <section
        className="category-list-section"
        aria-labelledby="category-list-title"
      >
        <div className="category-list-heading">
          <div>
            <h3 id="category-list-title">{t.categoriesManagement.list}</h3>
            <p>
              {t.categoriesManagement.count.replace(
                "{count}",
                String(categories.length),
              )}
            </p>
          </div>
          <label className="category-search">
            <span className="sr-only">{t.categoriesManagement.search}</span>
            <Search aria-hidden="true" />
            <Input
              value={query}
              placeholder={t.categoriesManagement.search}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
        </div>
        <ul className="category-manager-list">
          {visibleCategories.map((category) => (
            <li key={category.id}>
              <strong>{category.name}</strong>
              <code>{category.slug}</code>
            </li>
          ))}
        </ul>
        {!visibleCategories.length ? (
          <p className="empty-state-copy">{t.categoriesManagement.empty}</p>
        ) : null}
      </section>
    </div>
  );
}

function categorySlug(value: string) {
  const transliterated = value
    .toLocaleLowerCase()
    .replace(/[а-яё]/g, (letter) => cyrillic[letter] ?? "");
  return transliterated
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80);
}

const cyrillic: Record<string, string> = {
  а: "a", б: "b", в: "v", г: "g", д: "d", е: "e", ё: "e",
  ж: "zh", з: "z", и: "i", й: "y", к: "k", л: "l", м: "m",
  н: "n", о: "o", п: "p", р: "r", с: "s", т: "t", у: "u",
  ф: "f", х: "h", ц: "c", ч: "ch", ш: "sh", щ: "sch", ъ: "",
  ы: "y", ь: "", э: "e", ю: "yu", я: "ya",
};
