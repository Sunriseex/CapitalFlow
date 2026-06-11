# CapitalFlow Design Guide — shadcn neutral / popover-first UI

Status: active design contract  
Updated: 11 Jun 2026  
Source preview: `capitalflow-dashboard-popover-command-v7.html`  
Target product: CapitalFlow — self-hosted personal finance tracker / personal finance OS

This document replaces the older Nordic-only design direction. The product may keep a calm northern feeling, but the current UI direction is closer to **shadcn neutral**: restrained surfaces, compact layout, readable tables, command-style interactions, clear finance semantics, and minimal decoration.

The interface must feel like a practical daily finance tool, not a landing page, not a crypto dashboard, and not a marketing product.

---

## 1. Core product direction

CapitalFlow is a private, self-hosted finance application. Its interface must help the user quickly answer:

- How much money do I have?
- What changed recently?
- Which operations need attention?
- Which accounts, subscriptions, deposits, and savings goals are active?
- Why did a balance change?
- Can this financial record be audited later?

The UI must prioritize correctness and readability over visual novelty.

### Design values

1. **Calm** — no aggressive colors, no excessive animations, no visual noise.
2. **Exact** — money, dates, statuses, account names, and categories must be easy to scan.
3. **Fast** — frequent actions should be available through command menu, shortcuts, and compact popovers.
4. **Auditable** — detail views must explain where data came from and what changed.
5. **Predictable** — actions that affect money must not be hidden or surprising.
6. **Self-host friendly** — the UI must be useful without cloud branding, external services, or SaaS assumptions.

### What the UI is not

CapitalFlow is not:

- a marketing landing page;
- a banking clone with huge decorative hero sections;
- a crypto trading terminal;
- a gamified budget app;
- a glassmorphism showcase;
- a mobile-only finance app squeezed onto desktop.

---

## 2. Current approved visual direction

The approved direction is based on the v7 dashboard preview:

- left sidebar with optional collapsed state;
- compact topbar;
- command menu opened with `Ctrl+K` / `⌘K`;
- transaction search opened with a separate icon and `Ctrl+F` / `⌘F`;
- creation and detail flows opened as popover/dialog overlays;
- minimal icon-only theme and language controls in sidebar footer;
- light and dark themes only, no system option in the visible UI;
- shadcn-like neutral OKLCH tokens;
- cards, tables, and dialogs with restrained borders and subtle shadows;
- no heavy decorative effects.

### General look

Use:

- neutral background;
- white/light cards in light mode;
- dark slate-like cards in dark mode;
- subtle muted backgrounds for selected states;
- restrained badges for review states;
- compact rows and dense financial data;
- clear focus rings;
- minimal icon buttons.

Avoid:

- large gradients behind financial numbers;
- glassmorphism for core finance surfaces;
- heavy blur overlays except a very light command/dialog backdrop;
- scale animations on rows/cards;
- colorful dashboards with too many semantic colors;
- visible keyboard shortcut labels on every icon button.

---

## 3. App shell

The main shell uses a two-column desktop layout:

```text
┌───────────────┬──────────────────────────────────────────┐
│ Sidebar       │ Topbar                                   │
│               ├──────────────────────────────────────────┤
│ Navigation    │ Dashboard / page content                 │
│               │                                          │
│ Sidebar card  │ Main grid + right rail                   │
│               │                                          │
│ Theme/Language│ Dialogs / popovers open above this shell │
└───────────────┴──────────────────────────────────────────┘
```

### Desktop shell

Default desktop grid:

```css
.app-shell {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
}
```

Collapsed desktop grid:

```css
.app-shell.is-sidebar-collapsed {
  grid-template-columns: 76px minmax(0, 1fr);
}
```

Rules:

- Sidebar is visible by default on desktop.
- Sidebar can be collapsed by the topbar toggle button.
- Collapsed state is persisted in `localStorage`.
- Collapsing sidebar must not remove access to navigation.
- Sidebar collapse button must remain outside sidebar, because the sidebar may be collapsed.
- Main content must never jump horizontally in a way that causes data loss or input loss.
- Main content must use `min-width: 0` to prevent overflow in grids.

### Tablet behavior

Below `920px`:

- app becomes single-column;
- sidebar becomes a horizontal top section or future drawer;
- nav items scroll horizontally;
- collapsed sidebar state should not create a broken narrow column;
- topbar stacks vertically if needed;
- command trigger should become full width.

### Mobile behavior

Below `720px`:

- main padding becomes smaller;
- cards stack;
- desktop transaction table is hidden;
- mobile transaction cards are shown;
- dialogs use almost full width;
- action grids become one column;
- keyboard shortcuts remain active if a hardware keyboard is used.

---

## 4. Sidebar

The sidebar contains:

1. brand;
2. primary navigation;
3. optional sidebar card for review/import/status;
4. footer with icon-only controls.

### Brand

Expanded sidebar:

```text
[CF] CapitalFlow
     Personal finance
```

Collapsed sidebar:

```text
[CF]
```

Rules:

- Brand mark remains visible in collapsed mode.
- Brand text is hidden in collapsed mode.
- Brand should not be a large logo.
- Brand should link to overview/dashboard.

### Navigation

Navigation should be grouped by product logic, not by implementation folders.

Recommended primary navigation:

```text
Overview / Обзор
Accounts / Счета
Transactions / Операции
Transfers / Переводы
Goals / Цели
Deposits / Вклады
Subscriptions / Подписки
Analytics / Аналитика
Settings / Настройки
```

Future navigation candidates:

```text
Imports / Импорт
Review Queue / Проверка
Reconciliation / Сверка
Money Calendar / Календарь денег
Investments / Инвестиции
AI Assistant / AI-помощник
Backups / Резервные копии
Security / Безопасность
```

Rules:

- Current page uses `aria-current="page"`.
- Icon-only collapsed nav items must keep accessible labels.
- Avoid hiding core money actions behind nested menus too early.
- Do not add too many items before the feature exists.
- Settings and security can be separate later, but not necessary in the first compact UI.

### Sidebar footer

The footer contains only compact icon buttons:

```text
[☀/☾] [🇷🇺/🇬🇧]
```

Rules:

- Buttons are icon-only visually.
- Button size: `38px × 38px`.
- No visible labels like `Светлая`, `Тёмная`, `Русский`, `English` inside the footer buttons.
- Labels exist through `aria-label`, `title`, and screen-reader-only text.
- Theme icon:
  - light theme: sun icon `☀`;
  - dark theme: crescent/moon icon `☾`.
- Language icon:
  - current locale flag only;
  - on click opens a small popover.
- Language popover must show full language names in their own language:
  - `🇷🇺 Русский`;
  - `🇬🇧 English`.

Important: flags are visual hints, not locale identifiers. The persisted value must be locale code, for example:

```text
capitalflow_locale = ru | en
```

Theme persistence:

```text
capitalflow_theme = light | dark
```

Do not expose `system` theme in the current visible UI. Only `light` and `dark` are available.

---

## 5. Topbar

The topbar is the main quick-access area.

Approved order:

```text
[sidebar toggle] [command trigger] [transaction search icon] [Import CSV] [Add transaction] [notifications]
```

### Sidebar toggle

Purpose:

- collapse/expand sidebar on desktop;
- future mobile behavior can open a drawer.

Rules:

- Place the toggle before the command trigger.
- Use a compact icon-only button.
- Keep `aria-label` updated:
  - `Свернуть боковую панель`;
  - `Развернуть боковую панель`.
- Persist state in `localStorage`:

```text
capitalflow_sidebar_collapsed = true | false
```

### Command trigger

The command trigger is a wide button, not a normal search field.

Example:

```text
[⌕ Искать или выполнить команду...        ⌘K]
```

Rules:

- Open with `Ctrl+K` / `⌘K`.
- Use `aria-haspopup="dialog"`.
- Use `aria-keyshortcuts="Control+K Meta+K"`.
- It should search commands, navigation, and quick actions.
- It may include a few relevant transaction results, but it is not the full transaction search interface.

### Transaction search icon

A separate icon-only button near command trigger.

Rules:

- Visual appearance remains a simple icon button.
- Do not display `Ctrl+F` on the button.
- Use `aria-label="Поиск операций"`.
- Use `aria-keyshortcuts="Control+F Meta+F"`.
- Open transaction search dialog with click.
- `Ctrl+F` / `⌘F` opens transaction search instead of browser search when the app has focus.
- Do not conflict with command menu.
- If a text input or textarea already has focus, be careful before intercepting `Ctrl+F`. Preferred behavior:
  - if focus is inside transaction search, allow normal input search behavior;
  - if focus is inside another text input, do not destroy the user’s current input;
  - otherwise open app transaction search.

### Primary actions

Topbar primary actions:

```text
Import CSV
Add transaction
Notifications
```

Rules:

- `Add transaction` is the primary action.
- `Import CSV` is secondary.
- Notifications stay icon-only.
- Avoid placing too many buttons in topbar.
- On mobile, buttons may wrap, but the command trigger should remain usable.

---

## 6. Overlay interaction model

The current design is **popover-first**.

This means new elements should not expand into large permanent dashboard sections unless they are part of the current page content.

Use overlays for:

- command menu;
- transaction search;
- transaction detail;
- add account form;
- category picker/manager;
- empty states preview/onboarding;
- quick review of missed subscription;
- future quick create flows.

Do not use overlays for:

- full analytics pages;
- long CSV mapping workflows;
- complex settings pages;
- detailed import review with many rows;
- reports that need persistent filters.

### Overlay types

Use three levels:

1. **Command dialog** — global actions and navigation.
2. **Popover/dialog panel** — focused editing or detail task.
3. **Full page** — complex workflows that need space and persistent state.

### Shared overlay rules

Every overlay must:

- use `role="dialog"` where appropriate;
- use `aria-modal="true"` for modal dialogs;
- have a visible title;
- have an accessible description when helpful;
- trap focus;
- close with `Escape`;
- close via explicit close button;
- restore focus to the opener after close;
- avoid large layout shifts behind the overlay;
- not lose unsaved form data on accidental type switching.

Backdrop:

- light neutral overlay;
- minimal blur only if performance is acceptable;
- no heavy glass effects;
- no animated background.

---

## 7. Command menu

The command menu is opened through:

```text
Ctrl+K / ⌘K
```

Purpose:

- navigate quickly;
- run common actions;
- open focused dialogs;
- surface important review actions.

It is not a replacement for full transaction filtering.

### Command menu structure

Recommended groups:

```text
Actions / Действия
Navigation / Навигация
Reports / Отчёты
Review / Проверка
Recent / Недавнее
```

Example actions:

```text
Add transaction / Добавить операцию
Create transfer / Новый перевод
Add account / Добавить счёт
Import CSV / Импорт CSV
Open categories / Категории
Open empty states / Пустой старт
Review missed subscription / Проверить подписку
```

Example navigation:

```text
Overview
Accounts
Transactions
Transfers
Subscriptions
Analytics
Settings
```

### Command item anatomy

Each item:

```text
[icon] Title
       Description                         [optional key]
```

Rules:

- Title is short.
- Description explains what will happen.
- Icons are minimal text/SVG icons.
- Keyboard hint is optional.
- Do not use bright icons per item.
- Hidden items must not remain keyboard-focusable.

### Keyboard behavior

Required:

- `Ctrl+K` / `⌘K` toggles command menu.
- `Escape` closes.
- `ArrowDown` / `ArrowUp` moves through visible items.
- `Home` / `End` jump to first/last visible item.
- `Enter` activates focused item.
- `Tab` stays trapped inside the dialog.
- Focus returns to the command trigger or previous focused element on close.

### Search behavior

Search source:

- item title;
- item description;
- hidden keywords via `data-command-value`;
- route aliases;
- domain aliases.

Example hidden values:

```html
<button
  data-command-value="подписки recurring subscription regular payment spotify"
>
```

Search must be forgiving:

- Russian and English aliases may exist together.
- Common domain words should work: `карта`, `вклад`, `перевод`, `spotify`, `csv`, `категория`.

---

## 8. Transaction search

Transaction search is separate from command menu.

Open methods:

- search icon in topbar;
- `Ctrl+F` / `⌘F`.

### Why separate search exists

Command menu is for quick actions. Transaction search is for finding actual financial records.

Transaction search needs:

- query input;
- filters;
- result list;
- quick open into transaction detail;
- no permanent dashboard clutter.

### Transaction search dialog layout

```text
Search transactions
Description

[⌕ Find by merchant, category, account, amount...]

[This month] [Review] [Subscriptions] [Transfers] [Imported]

Results:
[merchant] [category/account/date] [amount/status]
[merchant] [category/account/date] [amount/status]
```

### Search fields

Must match:

- merchant/title;
- note;
- category;
- account;
- amount;
- currency;
- status;
- source;
- transfer/subscription relation;
- import batch metadata if available.

### Filters

Initial quick filters:

```text
This month / Этот месяц
Review / Проверить
Subscriptions / Подписки
Transfers / Переводы
Imported / Импортировано
```

Future filters:

```text
Date range
Account
Category
Type: income / expense / transfer / exchange
Status
Amount from/to
Source: manual / CSV / rule / generated
Currency
```

### Result item anatomy

```text
Merchant or operation title              −1 245 ₽
Category · Account · Date                Review badge
```

Rules:

- Amount is right-aligned.
- Negative sign must be visible.
- Do not rely on color alone.
- Clicking a result opens transaction detail dialog.
- `Enter` on focused result opens detail.

### Empty search state

Example:

```text
Операции не найдены.
Попробуйте изменить запрос или очистить фильтры.
```

Optional actions:

```text
[Очистить фильтры] [Добавить операцию]
```

---

## 9. Transactions list

The dashboard shows recent transactions. The full Transactions page will show advanced filtering.

### Desktop table

Required columns:

```text
Merchant / Note | Category | Account | Date | Status | Amount
```

For dashboard compact preview, acceptable columns:

```text
Merchant | Category | Account | Date | Amount
```

But the row must still show review/subscription/import status somewhere.

### Desktop row anatomy

```text
[icon] Merchant / operation title       Category       Account         Date        Amount
       Secondary metadata               Status/source
```

Example:

```text
Пятёрочка                               Продукты       T-Bank Black    Сегодня     −1 245 ₽
Вручную · Проверено
```

### Mobile transaction card

```text
Пятёрочка                               −1 245 ₽
Продукты · T-Bank Black
Сегодня, 14:20 · Проверено
```

Rules:

- Mobile uses cards, not squeezed tables.
- Desktop uses table for scanability.
- Rows are compact.
- Whole row/card can open detail.
- Hover only changes background subtly.
- Focus state must be visible.
- Do not use scale animations.

### Amount rules

Income:

```text
+120 000 ₽
```

Expense:

```text
−1 245 ₽
```

Transfer:

```text
−10 000 ₽ / +10 000 ₽
```

Exchange:

```text
−50 000 ₽ → +550 $
```

Rules:

- Use signs.
- Use currency.
- Use tabular numbers.
- Amounts in tables are right-aligned.
- Do not rely on red/green only.

### Status badges

Current initial badges:

```text
Проверить
Импортировано
Подписка
Перевод
Обмен валюты
Ошибка
Ожидается
Проверено
```

Badge rules:

- Use badges only when they add information.
- Normal completed manual expenses may not need a loud badge.
- Review/warning states should be visible but not alarming across the entire row.
- Error/danger badges must include text.
- Do not encode meaning by color alone.

---

## 10. Transaction detail dialog

Transaction detail is opened by:

- clicking transaction row/card;
- selecting transaction search result;
- command menu result/action;
- review alert action.

Purpose:

The detail view must explain:

- what happened;
- when it happened;
- which account changed;
- where data came from;
- whether it is linked to a subscription, transfer, import, or goal;
- what changed after creation.

### Detail layout

```text
Title / merchant                         [close]
Amount
Type · Category · Account · Date

[Action bar]

Main details
Source
Relations
Audit timeline
Raw import data / technical details if needed
```

### Header

Expense example:

```text
Пятёрочка
−1 245 ₽
Расход · Продукты · T-Bank Black · Сегодня, 14:20
```

Income example:

```text
Зарплата
+120 000 ₽
Доход · Зарплата · Основной счёт · 5 июня, 10:03
```

Transfer example:

```text
Пополнение резерва
−30 000 ₽ → +30 000 ₽
Перевод · Карта → Накопительный счёт
```

Exchange example:

```text
Обмен валюты
−50 000 ₽ → +550 $
Курс применён при создании операции
```

### Action bar

Default actions:

```text
Редактировать
Изменить категорию
Создать правило
Дублировать
Удалить
```

Imported transaction actions:

```text
Принять
Связать
Игнорировать
Исправить
```

Subscription transaction actions:

```text
Связать с подпиской
Создать подписку
Открыть подписку
Отметить как пропущенную
```

Transfer actions:

```text
Открыть связанную операцию
Показать business event
Показать курс
```

Rules:

- Destructive action is never primary.
- Delete action requires confirmation.
- Imported transaction actions depend on review state.
- Actions stay inside the detail dialog, not hidden in row hover only.

### Main details block

Fields:

```text
Тип
Категория
Счёт
Дата
Время
Сумма
Валюта
Статус
```

### Source block

Possible sources:

```text
Вручную
CSV импорт
Правило
Подписка
Сгенерировано системой
Корректировка
```

For CSV import:

```text
Источник: CSV импорт
Файл: tinkoff-june.csv
Строка: 42
Parser version: tinkoff-v1
Статус: Требует проверки / Проверено
```

Raw import data:

- hidden by default;
- available under disclosure/spoiler;
- never shown as the first thing in detail view.

### Relations block

Show only relevant relations.

Subscription:

```text
Подписка: Spotify
Периодичность: каждый месяц
Следующее списание: 10 июля 2026
Приоритет: optional / important / essential
```

Transfer:

```text
Business event: Transfer
Списано: −10 000 ₽ с T-Bank Black
Зачислено: +10 000 ₽ на Накопительный счёт
Связанные transaction legs: 2
```

Cross-currency transfer:

```text
Списано: −50 000 ₽
Зачислено: +550 $
Applied exchange rate: 90.91 ₽ / $
Rate source: manual / provider
Rate timestamp: 10 Jun 2026, 14:20
```

Goal allocation:

```text
Цель: Emergency fund
Тип: Пополнение цели
Режим: planning allocation
```

### Audit timeline

Compact timeline format:

```text
10 июн 14:25 · Создано вручную
10 июн 14:27 · Категория изменена: Прочее → Продукты
10 июн 14:28 · Связано с правилом “Супермаркеты”
```

Rules:

- Timeline must be concise.
- Use exact dates and times.
- Show user-visible actions first.
- Technical IDs may be hidden under details.
- This is not a replacement for backend audit model, only UI representation.

---

## 11. Category picker and category management

Categories should open in a command-like dialog, not as a plain select.

Use this for:

- changing transaction category;
- managing default categories;
- choosing category in transaction form;
- linking category to subscription logic.

### Category dialog layout

```text
Categories
Description

[⌕ Найти категорию...]
[All] [Income] [Expense] [Required] [Regular]

Category groups:
Income
Required expenses
Daily spending
Planning
Recurring payments
Personal
```

### Category option anatomy

```text
[icon] Category name
       Short explanation              [type/status badge]
```

Example:

```text
◇ Подписки
  Регулярные платежи и сервисы        Расход
```

### Default category groups

#### Income / Доходы

```text
Зарплата
Аванс
Премия
Фриланс
Проценты
Дивиденды
Подарки
Возврат
Продажа
Прочий доход
```

#### Required expenses / Обязательные расходы

```text
Жильё
Коммунальные услуги
Связь и интернет
Кредиты
Страховка
Налоги
Медицина
Образование
```

#### Daily spending / Повседневные расходы

```text
Продукты
Кафе и рестораны
Транспорт
Такси
Авто
Одежда
Бытовые товары
Аптека
Маркетплейсы
```

#### Planning / Финансовое планирование

```text
Накопления
Инвестиции
Вклад
Резервный фонд
Перевод между счетами
Обмен валюты
Комиссии
```

#### Recurring payments / Регулярные платежи

```text
Подписки
Сервисы
Игры
Музыка
Кино
Облако / хостинг
```

#### Personal / Личное

```text
Развлечения
Путешествия
Спорт
Подарки
Хобби
Домашние животные
Прочее
```

### Special rule: Subscriptions category

If user creates an expense with category `Подписки`, the app must not silently create a subscription.

Instead show a soft suggestion:

```text
Это похоже на регулярный платёж.
Создать подписку или связать с существующей?
[Создать подписку] [Связать] [Не сейчас]
```

Rules:

- Do not auto-create subscription without confirmation.
- Preserve the transaction even if user chooses `Не сейчас`.
- Allow linking to existing subscription.
- Yearly subscriptions should later show monthly equivalent for planning.
- Missed expected charge should create a review warning.

---

## 12. Add account dialog

Adding an account opens a focused dialog, not a long permanent dashboard section.

### Purpose

The form must adapt to account type. A card should not show interest fields. A savings account or deposit should show interest fields.

### Account type selector

Use selectable type cards instead of a raw dropdown.

Recommended types for MVP:

```text
Карта
Наличные
Текущий счёт
Накопительный счёт
Вклад
```

Future types:

```text
Инвестиционный счёт
Криптокошелёк
Брокерский счёт
Кредит
```

### Type card anatomy

```text
[icon] Type name
       One-sentence explanation
```

Examples:

```text
Карта
Для ежедневных расходов. Без процентных полей.
```

```text
Накопительный счёт
Для денег с процентами и свободным доступом.
```

```text
Вклад
Для суммы на срок с условиями пополнения и снятия.
```

### Shared fields

Shown for all account types:

```text
Название
Тип аккаунта
Валюта
Начальный баланс
Дата начального баланса
Включать в общий баланс
Описание / заметка
```

Rules:

- Labels are always visible.
- Placeholders do not replace labels.
- Currency must be explicit.
- Balance field must use money formatting.
- Do not ask for unnecessary fields in the first step.

### Card fields

For debit card:

```text
Название карты
Банк / источник
Валюта
Текущий баланс
Последние 4 цифры, необязательно
Кредитная карта? да/нет
Включать в общий баланс
```

Interest fields are hidden.

If `Кредитная карта = да`, reveal:

```text
Кредитный лимит
Льготный период, дней
Дата платежа
Минимальный платёж
```

### Cash fields

For cash:

```text
Название
Валюта
Текущий остаток
Место хранения, необязательно
Включать в общий баланс
```

Do not show:

- bank;
- interest rate;
- capitalization;
- card number;
- payment date.

### Checking account fields

For current/checking account:

```text
Название счёта
Банк / источник
Валюта
Текущий баланс
Account identifier, optional
Включать в общий баланс
```

Interest fields hidden by default. If later checking accounts can have interest, this should be an explicit advanced option.

### Savings account fields

For savings account, reveal interest block:

```text
Процентная ставка
Период начисления
Капитализация
Дата следующего начисления
Минимальный остаток, если есть
```

Rules:

- Savings account is still a liquid account.
- It is not the same as fixed deposit.
- Interest fields appear smoothly.
- User must understand whether money is freely available.

### Deposit fields

For deposit, reveal term and deposit conditions:

```text
Название вклада
Банк / источник
Валюта
Сумма открытия
Дата открытия
Дата окончания
Процентная ставка
Тип начисления процентов
Капитализация
Можно пополнять?
Можно частично снимать?
Что делать в конце срока?
```

Capitalization modes:

```text
Без капитализации
Ежедневная
Ежемесячная
В конце срока
```

Rules:

- Deposit is a term-based instrument.
- Always show opening date and end date.
- Always show refill/withdraw restrictions.
- Forecast and accrual history will be added later.

### Dynamic field behavior

When account type changes:

- do not instantly delete already typed hidden values;
- hide irrelevant fields;
- preserve temporary form state until submit or reset;
- show a small notice only if data might be hidden:

```text
Вы переключили тип счёта. Поля вклада скрыты, но данные сохранятся до сохранения формы.
```

Animation:

- subtle opacity/height transition;
- no bounce;
- respect `prefers-reduced-motion`.

---

## 13. Empty states

Empty states are available through a dialog preview in the design mockup, but in the real app they appear inside the relevant page.

### Empty state structure

```text
[small icon]
Title
One sentence explaining what is missing
Primary action
Optional secondary action
```

### Empty accounts

```text
Счетов пока нет
Создайте карту, наличные или накопительный счёт, чтобы начать учёт.
[Добавить счёт] [Импортировать CSV]
```

### Empty transactions

```text
Операций пока нет
Добавьте первую операцию вручную или импортируйте выписку. После импорта строки сначала попадут на проверку.
[Добавить операцию] [Импорт CSV]
```

### Empty subscriptions

```text
Подписок пока нет
Когда расход будет отмечен категорией “Подписки”, приложение предложит создать регулярный платёж.
[Открыть категории]
```

### Empty review queue

```text
Нет задач на проверку
Это нормальное состояние. Проверка появится только для импорта, дубликатов, пропущенных подписок или ошибок курса.
```

Rules:

- Do not write vague empty states like `Nothing here`.
- Empty state must tell the user what to do next.
- Avoid large illustrations.
- Empty state should not look like an error.

---

## 14. Cards

Cards are used for metrics, accounts, budgets, subscriptions, right rail widgets, and small summaries.

### Card anatomy

```text
Card
  Header
    Title
    Description
    Optional action/badge
  Content
```

Rules:

- Use cards for grouped information, not every tiny label.
- Keep card headers compact.
- Do not make every card visually equal if one is more important.
- Metric values should be large but not hero-sized.
- Avoid decorative gradients behind critical numbers.

### Metric cards

Metric card fields:

```text
Label
Short description
Main value
Subtext / delta
Optional badge
```

Examples:

```text
Общий баланс
По всем активным счетам
742 300 ₽
+34 200 ₽ за месяц
```

```text
Подписки
Месячный эквивалент
4 590 ₽
1 требует проверки
```

### Account cards

Account item fields:

```text
Account name
Type / usage
Balance
Optional secondary metric
```

Examples:

```text
T-Bank Black
Карта для ежедневных расходов
25 400 ₽
```

```text
Emergency fund
Накопительный счёт · 14%
180 000 ₽
```

Rules:

- Show balance clearly.
- Show account type.
- For savings account, APY/rate may be secondary.
- For card, do not show APY unless it is actually a savings/card product.

---

## 15. Tables and dense data

Tables remain important on desktop.

### Transaction table rules

- Use real table markup for tabular data.
- Keep rows compact: `44px` to `56px` when possible.
- Amount column right-aligned.
- Date/status easy to scan.
- Header can become sticky on full Transactions page.
- Full table may horizontally scroll only as a last resort.
- On mobile, switch to card list.

### Table hover/focus

- Hover: muted background.
- Focus: visible ring or row outline.
- No scale.
- No row shadow.

### Sorting/filtering

Full Transactions page should support:

```text
Search
Date range
Account
Category
Status
Type
Amount range
Source
Sort by date
Sort by amount
Pagination or infinite loading
```

Dashboard recent transactions should stay simpler.

---

## 16. Forms

### Field structure

```text
Label
Input / select / custom picker
Hint or error
```

Rules:

- Labels are always visible.
- Placeholder is only a hint.
- Error appears near the field.
- Server/API error appears in alert region above form.
- Submit button disabled until valid, but user must understand why.
- Money fields show currency.
- Date fields use localized date format.

### Form width

- Dialog forms: `640px` to `760px` max depending on complexity.
- Simple forms: `480px` to `640px`.
- Wide dialogs only when there are two columns, such as account type + form fields.

### Form validation

Validate:

- on blur;
- on submit;
- optionally as user types for money/date fields.

Do not show all errors before user interacts.

---

## 17. Theme system

Current visible theme options:

```text
light
dark
```

No visible `system` option.

### Theme storage

```text
localStorage key: capitalflow_theme
values: light | dark
```

### Theme toggle behavior

- icon-only in sidebar footer;
- sun icon for light theme;
- crescent/moon icon for dark theme;
- click toggles theme;
- update `aria-pressed`;
- update `aria-label`:
  - if current is light: `Переключить на тёмную тему`;
  - if current is dark: `Переключить на светлую тему`.

### Theme implementation

Preferred:

```html
<html data-theme="light">
```

or:

```html
<html data-theme="dark">
```

Do not rely only on `prefers-color-scheme` after user explicitly chooses a theme.

---

## 18. Language system

Current visible languages:

```text
ru
en
```

### Language storage

```text
localStorage key: capitalflow_locale
values: ru | en
```

### Language toggle

- icon-only in sidebar footer;
- shows current locale flag only;
- click opens popover;
- popover uses language names in their own language:

```text
🇷🇺 Русский
🇬🇧 English
```

### Language popover behavior

- role: menu or listbox is acceptable, but be consistent;
- current language visually selected;
- current language has `aria-selected="true"` or radio-like state;
- close on outside click;
- close on `Escape`;
- return focus to language button.

### Translation rules

- UI strings must go through i18n dictionary.
- Do not translate persisted user data automatically.
- Category names seeded by system may have localized display names.
- User-created category names remain as the user entered them.
- Dates and money use locale-aware formatting helpers.

---

## 19. Money formatting

Use a single helper for all money values.

Rules:

- Always show currency.
- Use signs for deltas and transaction amounts.
- Use locale-aware grouping.
- Use tabular numbers.
- Align money values to the right in tables.
- Do not mix `RUB`, `₽`, and `руб.` randomly in the same screen.

Recommended Russian compact display:

```text
128 450 ₽
+12 300 ₽
−4 990 ₽
```

Recommended detailed display if decimals matter:

```text
128 450,20 ₽
+12 300,00 ₽
−4 990,00 ₽
```

Recommended CSS:

```css
.money,
.metric-value,
.table-amount {
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
}
```

Rules for crypto/stocks later:

- do not force 2 decimals for assets requiring higher precision;
- preserve original asset precision;
- display conversion separately from persisted financial truth.

---

## 20. Date and time formatting

Rules:

- Use localized date formatting.
- Recent transaction list may use relative labels:
  - `Сегодня`;
  - `Вчера`;
  - `10 июн`.
- Detail view must show exact date and time.
- Imported data should preserve original timestamp if available.
- Audit timeline must use exact timestamp.

Examples:

```text
Сегодня, 14:20
10 июня 2026, 14:20
10 июн 14:25 · Создано вручную
```

---

## 21. Status, warnings, and review states

CapitalFlow needs explicit states because financial data can be imported, generated, linked, edited, or suspicious.

### Common statuses

```text
Проверено
Проверить
Импортировано
Ожидается
Ошибка
Пропущено
Связано
Черновик
```

### Domain badges

```text
Подписка
Перевод
Обмен валюты
Цель
Вклад
Корректировка
```

### Warning patterns

Use alert blocks for important issues:

```text
Ожидаемое списание подписки не найдено
Spotify ожидался 7 июня. Подтвердите платёж вручную или поставьте подписку на паузу.
[Проверить]
```

Rules:

- Warning must include action.
- Warning should not block the whole dashboard unless critical.
- Red is for destructive/error states, not every expense.
- Amber is for review/attention.
- Green is for success/income.

---

## 22. Accessibility

Minimum requirements:

- All icon-only buttons have `aria-label`.
- All dialogs have title and close button.
- Modals trap focus.
- `Escape` closes overlays.
- Focus returns to opener.
- All inputs have labels.
- Error messages are connected to fields where possible.
- Use real tables for tabular desktop data.
- Do not rely on color alone.
- Use visible `:focus-visible`.
- Respect `prefers-reduced-motion`.

Recommended focus style:

```css
:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}
```

### Keyboard shortcuts

Current shortcuts:

```text
Ctrl+K / ⌘K — command menu
Ctrl+F / ⌘F — transaction search
Escape — close active overlay
ArrowUp/ArrowDown — navigate command/search results
Enter — open selected item
```

Rules:

- Shortcuts must not break text input editing.
- Shortcuts must not be the only way to access a feature.
- Visible button always exists for command/search.

---

## 23. Motion and performance

Allowed:

- subtle color transition on hover;
- subtle opacity/height reveal for conditional form blocks;
- short dialog enter/exit transition if implemented later;
- no animation when `prefers-reduced-motion: reduce`.

Avoid:

- SVG turbulence/noise filters;
- heavy backdrop filters;
- scale animations on financial cards/rows;
- parallax;
- infinite animated backgrounds;
- skeleton shimmer if it causes performance issues.

Reduced motion rule:

```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    scroll-behavior: auto !important;
    transition: none !important;
    animation-duration: 0.001ms !important;
    animation-iteration-count: 1 !important;
  }
}
```

---

## 24. Design tokens

The current v7 preview uses shadcn-like OKLCH tokens.

### Base tokens

```css
:root {
  --radius: 0.625rem;

  --background: oklch(1 0 0);
  --foreground: oklch(0.145 0 0);

  --card: oklch(1 0 0);
  --card-foreground: oklch(0.145 0 0);

  --popover: oklch(1 0 0);
  --popover-foreground: oklch(0.145 0 0);

  --primary: oklch(0.205 0 0);
  --primary-foreground: oklch(0.985 0 0);

  --secondary: oklch(0.97 0 0);
  --secondary-foreground: oklch(0.205 0 0);

  --muted: oklch(0.97 0 0);
  --muted-foreground: oklch(0.556 0 0);

  --accent: oklch(0.97 0 0);
  --accent-foreground: oklch(0.205 0 0);

  --destructive: oklch(0.577 0.245 27.325);
  --destructive-foreground: oklch(0.985 0 0);

  --success: oklch(0.53 0.12 145);
  --warning: oklch(0.65 0.16 70);

  --border: oklch(0.922 0 0);
  --input: oklch(0.922 0 0);
  --ring: oklch(0.708 0 0);

  --radius-sm: calc(var(--radius) * 0.6);
  --radius-md: calc(var(--radius) * 0.8);
  --radius-lg: var(--radius);
  --radius-xl: calc(var(--radius) * 1.4);

  --shadow-sm: 0 1px 2px rgb(0 0 0 / 0.05);
  --shadow-md: 0 18px 60px rgb(0 0 0 / 0.16);

  --font-sans: Inter, ui-sans-serif, system-ui, -apple-system,
    BlinkMacSystemFont, "Segoe UI", sans-serif;
}
```

### Dark tokens

```css
html[data-theme="dark"] {
  --background: oklch(0.145 0 0);
  --foreground: oklch(0.985 0 0);

  --card: oklch(0.205 0 0);
  --card-foreground: oklch(0.985 0 0);

  --popover: oklch(0.205 0 0);
  --popover-foreground: oklch(0.985 0 0);

  --primary: oklch(0.922 0 0);
  --primary-foreground: oklch(0.205 0 0);

  --secondary: oklch(0.269 0 0);
  --secondary-foreground: oklch(0.985 0 0);

  --muted: oklch(0.269 0 0);
  --muted-foreground: oklch(0.708 0 0);

  --accent: oklch(0.269 0 0);
  --accent-foreground: oklch(0.985 0 0);

  --destructive: oklch(0.704 0.191 22.216);
  --destructive-foreground: oklch(0.985 0 0);

  --success: oklch(0.72 0.14 145);
  --warning: oklch(0.76 0.14 75);

  --border: oklch(1 0 0 / 10%);
  --input: oklch(1 0 0 / 15%);
  --ring: oklch(0.556 0 0);

  --shadow-sm: none;
}
```

### Token rules

- Use semantic tokens in components.
- Do not hardcode colors unless adding a new token.
- Add new tokens only if repeated across multiple components.
- Keep finance semantic colors restrained.
- Dark theme must not be pure black.

---

## 25. Typography

Use system sans-serif with Inter preferred.

```css
body {
  font-family: Inter, ui-sans-serif, system-ui, -apple-system,
    BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 14px;
  line-height: 1.5;
}
```

Recommended sizes:

```text
Page title: 24–32px / 1.1 / 600
Section title: 14–18px / 1.2 / 600
Card title: 14px / 1.2 / 600
Body: 14px / 1.5 / 400
Small text: 12–13px / 1.5 / 400–500
Metric value: 24–32px / 1 / 600
```

Rules:

- Avoid huge app-screen hero text.
- Use negative letter spacing only for large titles/metrics.
- Use tabular numbers for money and charts.
- Keep muted text readable.

---

## 26. Spacing, radius, and shadows

Current rhythm:

```text
4px micro gap
8px control gap
12px compact section gap
16px sidebar/card inner rhythm
18px current page gap in preview
24px larger page/section gap when needed
```

Recommended implementation tokens:

```css
:root {
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
}
```

Radius:

- normal controls: `var(--radius-md)`;
- cards: `var(--radius-xl)` in current preview;
- dialogs: `var(--radius-xl)`;
- badges: `999px`.

Shadows:

- light mode: subtle shadows allowed;
- dark mode: shadows can be disabled or minimized;
- no dramatic card elevation.

---

## 27. Loading, error, and success states

### Loading

Use skeletons:

- metric cards: skeleton blocks;
- transaction table: 5–8 skeleton rows;
- chart: placeholder with title retained;
- dialogs: keep structure visible.

Avoid full-page spinner except during app boot.

### Error

Bad:

```text
Something went wrong
```

Better:

```text
Не удалось загрузить операции.
Проверьте соединение или повторите попытку.
[Повторить]
```

Rules:

- Error should describe the failed thing.
- Provide recovery action.
- Show field-specific errors near fields.
- Show API/server errors above form.

### Success

For routine finance actions:

```text
Операция добавлена
Баланс обновлён.
```

Rules:

- Keep success restrained.
- Avoid large animations.
- Confirm what changed.

---

## 28. Dashboard composition

Dashboard is an overview, not a workspace for every task.

Recommended order:

```text
Topbar
Attention alert, only if needed
Metrics grid
Main grid:
  Left:
    Spending overview / cash flow chart
    Recent transactions
  Right:
    Quick actions
    Accounts
    Budget status
    Upcoming subscriptions
```

Rules:

- Dashboard should answer high-level questions.
- Deep editing opens popovers/dialogs.
- Full workflows open dedicated pages.
- Alerts appear only when action is needed.
- Keep recent transactions compact.

---

## 29. Charts

Charts should support decisions, not decorate.

Rules:

- Every chart has title, period, units, and summary metric.
- Every chart has accessible text/table alternative.
- Use line charts for balances over time.
- Use bars for income vs expense.
- Use progress lists for budgets/categories.
- Avoid donut charts unless they are clearly better.
- Avoid 3D, heavy gradients, and tiny unlabeled slices.

Chart summary example:

```text
Движение денег
Последние 12 месяцев
Расходы: 1 248 ₽ за текущий месяц
Среднее: 1 159 ₽ / месяц
```

---

## 30. Implementation priority

### PR 1 — update design tokens and app shell

Scope:

- shadcn neutral tokens;
- light/dark theme via `data-theme`;
- AppShell;
- collapsible sidebar;
- topbar with sidebar toggle;
- command trigger;
- transaction search icon;
- sidebar footer icon-only theme/language controls.

Do not change backend API.

### PR 2 — command menu and transaction search

Scope:

- command dialog;
- `Ctrl+K` / `⌘K`;
- transaction search dialog;
- `Ctrl+F` / `⌘F`;
- focus trap;
- search filtering;
- keyboard navigation;
- empty result states.

### PR 3 — transaction list and detail dialog

Scope:

- desktop transaction table;
- mobile transaction cards;
- review/status badges;
- detail dialog;
- action bar;
- source/relations/audit timeline blocks.

### PR 4 — account creation dialog

Scope:

- account type picker cards;
- shared fields;
- card fields;
- credit card conditional fields;
- savings account interest fields;
- deposit term/interest fields;
- hidden field preservation during type switch.

### PR 5 — categories dialog and subscription prompt

Scope:

- default categories seed/display;
- category picker command-style dialog;
- category search;
- category groups;
- special `Подписки` suggestion flow.

### PR 6 — empty states and review UX

Scope:

- accounts empty state;
- transactions empty state;
- subscriptions empty state;
- review queue empty state;
- missed subscription alert;
- import review summary placeholder.

---

## 31. First implementation prompt for Codex

Use this as the next coding prompt:

```text
Update the frontend design foundation to match the current CapitalFlow v7 design direction.

Goal:
Implement a shadcn-neutral, popover-first app shell with compact finance UI.

Scope:
1. Add/align design tokens for light and dark themes using semantic CSS variables.
2. Implement AppShell with sidebar, collapsible sidebar state, topbar, and main content area.
3. Add icon-only sidebar footer controls for theme and language.
4. Add theme switching: light/dark only, persisted in localStorage.
5. Add language popover: flag-only trigger, choices shown as 🇷🇺 Русский and 🇬🇧 English.
6. Move transaction search icon next to the command trigger.
7. Add Ctrl+K / Meta+K for command menu.
8. Add Ctrl+F / Meta+F for transaction search, without changing the search icon visually.
9. Ensure overlays trap focus, close with Escape, and restore focus.
10. Keep dashboard compact; new details/forms must open as dialogs/popovers, not as permanent page sections.

Constraints:
- Do not change backend API.
- Do not rewrite unrelated frontend code.
- Do not add heavy UI libraries.
- Keep TypeScript strict.
- Keep components small.
- Use semantic HTML.
- Icon-only buttons must have aria-label.
- Respect prefers-reduced-motion.
```

---

## 32. Review checklist

Before merging UI changes, check:

### Layout

- [ ] Sidebar works expanded and collapsed.
- [ ] Topbar does not overflow at desktop/tablet widths.
- [ ] Mobile layout stacks correctly.
- [ ] Dashboard remains compact.
- [ ] New workflows open as dialog/popover, not as permanent dashboard blocks.

### Shortcuts

- [ ] `Ctrl+K` opens command menu.
- [ ] `Ctrl+F` opens transaction search.
- [ ] Browser search is not accidentally triggered in normal app state.
- [ ] Shortcuts do not break typing in active inputs.

### Dialogs

- [ ] Focus is trapped.
- [ ] Escape closes.
- [ ] Close button works.
- [ ] Focus returns to opener.
- [ ] Dialog has title and description.

### Transactions

- [ ] Desktop uses table.
- [ ] Mobile uses cards.
- [ ] Amounts have signs and currency.
- [ ] Review statuses are visible.
- [ ] Detail dialog shows source, relations, and audit timeline.

### Account form

- [ ] Card does not show interest fields.
- [ ] Savings account shows interest fields.
- [ ] Deposit shows term and capitalization fields.
- [ ] Credit card fields appear only when enabled.
- [ ] Switching type does not silently delete entered data.

### Theme/language

- [ ] Theme button is icon-only.
- [ ] Language button is icon-only.
- [ ] Language popover shows full names.
- [ ] Theme persists.
- [ ] Locale persists.
- [ ] Dark theme has readable contrast.

### Accessibility

- [ ] Icon-only buttons have aria-label.
- [ ] Forms have visible labels.
- [ ] Tables use semantic markup.
- [ ] Focus ring is visible.
- [ ] Color is not the only meaning.
- [ ] Reduced motion is respected.

### Performance

- [ ] No heavy SVG filters.
- [ ] No unnecessary backdrop-filter stacks.
- [ ] No infinite decorative animation.
- [ ] Large tables are not rendered without pagination/virtualization later.

---

## 33. Decisions locked by this document

These decisions should not be changed casually:

1. The current UI direction is shadcn neutral, not old Nordic glass/gradient UI.
2. Dashboard remains compact.
3. New detail/create flows open as popovers/dialogs.
4. Command menu is for commands and navigation.
5. Transaction search is separate and opens via icon + `Ctrl+F`.
6. Search icon remains visually just an icon.
7. Sidebar is collapsible on desktop.
8. Theme/language controls live in sidebar footer as icon-only buttons.
9. Theme options are only light/dark.
10. Language choices show full language names inside the popover.
11. Transaction detail must show source, relations, and audit timeline.
12. Account form fields depend on account type.
13. Card accounts do not show interest fields.
14. Savings accounts and deposits are different UI/domain concepts.
15. Category `Подписки` triggers a suggestion flow, not silent subscription creation.

