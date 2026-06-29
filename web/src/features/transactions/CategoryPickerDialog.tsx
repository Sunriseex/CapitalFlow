import { useMemo, useState } from "react";
import { BadgeCheck, Circle, Tag } from "lucide-react";
import type { Category } from "../../api/types";
import { useI18n } from "../../shared/i18n/useI18n";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../../components/ui/command";
import { Button as ShadcnButton } from "../../components/ui/button";
import { CategoryBadge } from "./components/CategoryBadge";

type CategoryFilter = "all" | "income" | "expense" | "required" | "regular";

export function CategoryPickerDialog({
  categories,
  selectedCategoryId,
  onSelect,
  onClose,
}: {
  categories: Category[];
  selectedCategoryId: string;
  onSelect: (categoryId: string) => void;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [filter, setFilter] = useState<CategoryFilter>("all");
  const groups = useMemo(() => categoryGroups(t), [t]);
  const groupedCategories = useMemo(
    () => groupCategories(categories, groups, filter),
    [categories, filter, groups],
  );

  return (
    <CommandDialog
      open
      title={t.transactions.categoryPickerTitle}
      description={t.transactions.categoryPickerDescription}
      className="category-picker-dialog"
      showCloseButton
      onOpenChange={(open) => !open && onClose()}
    >
      <div className="category-picker-layout">
        <aside
          className="category-search-panel"
          aria-label={t.transactions.categoryPickerActions}
        >
          <CommandInput placeholder={t.transactions.categoryPickerPlaceholder} />
          <div className="category-picker-filters" role="group">
            {(["all", "income", "expense", "required", "regular"] as const).map(
              (value) => (
                <ShadcnButton
                  key={value}
                  className={
                    filter === value ? "filter-chip is-active" : "filter-chip"
                  }
                  type="button"
                  variant="ghost"
                  aria-pressed={filter === value}
                  onClick={() => setFilter(value)}
                >
                  {t.transactions.categoryFilters[value]}
                </ShadcnButton>
              ),
            )}
          </div>
          <div className="preview-note">
            <strong>{t.transactions.subscriptionPromptTitle}</strong>
            <p>{t.transactions.subscriptionPromptDescription}</p>
          </div>
        </aside>
        <CommandList className="category-picker-list" role="listbox">
          <CommandEmpty>{t.transactions.categoryPickerEmpty}</CommandEmpty>
          <CommandGroup heading={t.transactions.categoryPickerActions}>
            <CommandItem
              className="category-option"
              value={`${t.common.none} no category uncategorized`}
              onSelect={() => {
                onSelect("");
                onClose();
              }}
            >
              <Circle aria-hidden="true" />
              <span className="category-option-copy">
                <strong>{t.common.none}</strong>
                <small>{t.transactions.noCategoryDescription}</small>
              </span>
              {!selectedCategoryId ? <BadgeCheck aria-hidden="true" /> : null}
            </CommandItem>
          </CommandGroup>
          {groupedCategories.map((group) => (
            <CommandGroup key={group.title} heading={group.title}>
              {group.categories.map((category) => (
                <CommandItem
                  className="category-option"
                  key={category.id}
                  value={`${category.name} ${category.slug} ${group.title}`}
                  onSelect={() => {
                    onSelect(category.id);
                    onClose();
                  }}
                >
                  <Tag aria-hidden="true" />
                  <span className="category-option-copy">
                    <CategoryBadge
                      categoryKey={category.id}
                      name={category.name}
                    />
                    <small>{group.description}</small>
                  </span>
                  <span className="tag muted">{group.badge}</span>
                  {selectedCategoryId === category.id ? (
                    <BadgeCheck aria-hidden="true" />
                  ) : null}
                </CommandItem>
              ))}
            </CommandGroup>
          ))}
        </CommandList>
      </div>
    </CommandDialog>
  );
}

function groupCategories(
  categories: Category[],
  groups: ReturnType<typeof categoryGroups>,
  filter: CategoryFilter,
) {
  const used = new Set<string>();
  const result = groups
    .filter((group) => filter === "all" || group.filter === filter)
    .map((group) => {
      const groupCategories = categories.filter((category) => {
        if (used.has(category.id)) {
          return false;
        }

        const matched = group.names.some((name) =>
          normalized(category.name, category.slug).includes(normalize(name)),
        );
        if (matched) {
          used.add(category.id);
        }
        return matched;
      });
      return { ...group, categories: groupCategories };
    })
    .filter((group) => group.categories.length > 0);

  if (filter === "all") {
    const uncategorized = categories.filter((category) => !used.has(category.id));
    if (uncategorized.length > 0) {
      result.push({
        key: "other",
        title: groups.at(-1)?.fallbackTitle ?? "Other",
        description: groups.at(-1)?.fallbackDescription ?? "",
        badge: groups.at(-1)?.fallbackBadge ?? "",
        filter: "expense",
        names: [],
        categories: uncategorized,
      });
    }
  }

  return result;
}

function categoryGroups(t: ReturnType<typeof useI18n>["t"]) {
  const labels = t.transactions.categoryGroups;
  return [
    {
      key: "income",
      title: labels.income,
      description: t.transactions.categoryGroupDescriptions.income,
      badge: t.transactions.categoryFilters.income,
      filter: "income" as const,
      names: [
        "salary",
        "зарплата",
        "advance",
        "аванс",
        "bonus",
        "премия",
        "freelance",
        "фриланс",
        "interest",
        "проценты",
        "dividend",
        "дивиденды",
        "gift",
        "подарки",
        "refund",
        "возврат",
        "sale",
        "продажа",
      ],
    },
    {
      key: "required",
      title: labels.required,
      description: t.transactions.categoryGroupDescriptions.required,
      badge: t.transactions.categoryFilters.required,
      filter: "required" as const,
      names: [
        "housing",
        "жиль",
        "utilities",
        "коммун",
        "internet",
        "связь",
        "credit",
        "кредит",
        "insurance",
        "страх",
        "tax",
        "налог",
        "medical",
        "медиц",
        "education",
        "образ",
      ],
    },
    {
      key: "daily",
      title: labels.daily,
      description: t.transactions.categoryGroupDescriptions.daily,
      badge: t.transactions.categoryFilters.expense,
      filter: "expense" as const,
      names: [
        "groceries",
        "продукт",
        "restaurant",
        "кафе",
        "transport",
        "транспорт",
        "taxi",
        "такси",
        "auto",
        "авто",
        "clothes",
        "одеж",
        "pharmacy",
        "аптек",
        "marketplace",
        "маркет",
      ],
    },
    {
      key: "planning",
      title: labels.planning,
      description: t.transactions.categoryGroupDescriptions.planning,
      badge: t.transactions.categoryFilters.expense,
      filter: "expense" as const,
      names: [
        "saving",
        "накоп",
        "investment",
        "инвест",
        "deposit",
        "вклад",
        "reserve",
        "резерв",
        "transfer",
        "перевод",
        "exchange",
        "обмен",
        "fee",
        "комисс",
      ],
    },
    {
      key: "regular",
      title: labels.regular,
      description: t.transactions.categoryGroupDescriptions.regular,
      badge: t.transactions.categoryFilters.regular,
      filter: "regular" as const,
      names: [
        "subscription",
        "подпис",
        "service",
        "сервис",
        "games",
        "игры",
        "music",
        "музык",
        "movie",
        "кино",
        "cloud",
        "облако",
        "hosting",
        "хост",
      ],
    },
    {
      key: "personal",
      title: labels.personal,
      description: t.transactions.categoryGroupDescriptions.personal,
      badge: t.transactions.categoryFilters.expense,
      filter: "expense" as const,
      fallbackTitle: labels.personal,
      fallbackDescription: t.transactions.categoryGroupDescriptions.personal,
      fallbackBadge: t.transactions.categoryFilters.expense,
      names: [
        "entertainment",
        "развлеч",
        "travel",
        "путеш",
        "sport",
        "спорт",
        "hobby",
        "хобби",
        "pets",
        "живот",
        "other",
        "проч",
      ],
    },
  ];
}

function normalized(...values: string[]) {
  return values.map(normalize).join(" ");
}

function normalize(value: string) {
  return value.trim().toLocaleLowerCase();
}
