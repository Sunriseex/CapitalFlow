package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const Email = "demo@capitalflow.local"

var namespace = uuid.MustParse("76051383-15d1-4e7d-93a9-d9e11ab45d08")

type Result struct {
	Accounts     int
	Transactions int
	Transfers    int
	Accruals     int
	Goals        int
	Limits       int
}

type transactionSeed struct {
	id, accountID, transactionType, amount, description string
	relatedAccountID, transferID, categoryID            *string
	occurredAt, createdAt                               time.Time
}

type transferSeed struct {
	id, fromAccountID, toAccountID, fromTransactionID, toTransactionID string
	amount, currency                                                   string
	createdAt                                                          time.Time
}

type accrualSeed struct {
	id, accountID, ruleID, transactionID, amount, balance string
	date, createdAt                                       time.Time
}

type dataset struct {
	transactions []transactionSeed
	transfers    []transferSeed
	accruals     []accrualSeed
	balances     map[string]decimal.Decimal
}

func ValidateEnvironment(appEnv string) error {
	if appEnv != "development" {
		return fmt.Errorf("demo seed requires APP_ENV=development")
	}
	return nil
}

func Seed(ctx context.Context, pool *pgxpool.Pool, passwordHash string, now time.Time) (Result, error) {
	if passwordHash == "" {
		return Result{}, fmt.Errorf("demo password hash is required")
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Result{}, fmt.Errorf("begin demo seed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		return Result{}, fmt.Errorf("defer demo constraints: %w", err)
	}
	userID := seedID("user")
	if err := resetTx(ctx, tx, userID); err != nil {
		return Result{}, err
	}
	createdAt := dateOnly(now).AddDate(-6, 0, 0)
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, primary_currency, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, 'RUB', $4, $4, $4)
	`, userID, Email, passwordHash, createdAt); err != nil {
		return Result{}, fmt.Errorf("insert demo user: %w", err)
	}

	categories, err := ensureCategories(ctx, tx, createdAt)
	if err != nil {
		return Result{}, err
	}
	accounts := demoAccounts(createdAt)
	for _, account := range accounts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO accounts (id, name, bank, type, currency, is_active, opened_at, owner_user_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, true, $6, $7, $6, $6)
		`, account.id, account.name, account.bank, account.kind, account.currency, createdAt, userID); err != nil {
			return Result{}, fmt.Errorf("insert demo account %s: %w", account.name, err)
		}
	}

	ruleID := seedID("interest-rule:deposit")
	if _, err := tx.Exec(ctx, `
		INSERT INTO interest_rules
			(id, account_id, annual_rate_bps, accrual_frequency, capitalization_frequency, day_count_convention, is_active, start_date, created_at, updated_at)
		VALUES ($1, $2, 1200, 'monthly', 'monthly', 'actual_365', true, $3, $4, $4)
	`, ruleID, seedID("account:deposit"), createdAt, createdAt); err != nil {
		return Result{}, fmt.Errorf("insert demo interest rule: %w", err)
	}

	data := buildDataset(now, categories, ruleID)
	if err := insertTransfers(ctx, tx, userID, data.transfers); err != nil {
		return Result{}, err
	}
	if err := insertTransactions(ctx, tx, data.transactions); err != nil {
		return Result{}, err
	}
	if err := insertAccruals(ctx, tx, data.accruals); err != nil {
		return Result{}, err
	}
	if err := insertGoalsAndLimits(ctx, tx, userID, now, data.balances, categories); err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit demo seed: %w", err)
	}
	return Result{Accounts: len(accounts), Transactions: len(data.transactions), Transfers: len(data.transfers), Accruals: len(data.accruals), Goals: 4, Limits: 4}, nil
}

func Reset(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin demo reset: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		return fmt.Errorf("defer demo reset constraints: %w", err)
	}
	if err := resetTx(ctx, tx, seedID("user")); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit demo reset: %w", err)
	}
	return nil
}

func resetTx(ctx context.Context, tx pgx.Tx, userID string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM transfers WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete demo transfers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM transactions WHERE account_id IN (SELECT id FROM accounts WHERE owner_user_id = $1)`, userID); err != nil {
		return fmt.Errorf("delete demo transactions: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1 OR lower(email) = lower($2)`, userID, Email); err != nil {
		return fmt.Errorf("delete demo user: %w", err)
	}
	return nil
}

type accountSeed struct{ id, name, bank, kind, currency string }

func demoAccounts(_ time.Time) []accountSeed {
	return []accountSeed{
		{seedID("account:checking"), "Основной счёт", "T-Bank", "cash", "RUB"},
		{seedID("account:card"), "Карта расходов", "T-Bank", "card", "RUB"},
		{seedID("account:savings"), "Резервный фонд", "Yandex", "savings", "RUB"},
		{seedID("account:deposit"), "Срочный вклад", "Sber", "term_deposit", "RUB"},
		{seedID("account:usd"), "USD накопления", "Raiffeisen", "savings", "USD"},
	}
}

func ensureCategories(ctx context.Context, tx pgx.Tx, now time.Time) (map[string]string, error) {
	names := map[string]string{
		"salary": "Зарплата", "food": "Продукты", "transport": "Транспорт",
		"subscriptions": "Подписки", "housing": "Жильё", "health": "Здоровье",
		"entertainment": "Развлечения", "deposit_interest": "Проценты по вкладам",
		"other": "Прочее", "travel": "Путешествия", "clothing": "Одежда",
		"gifts": "Подарки", "education": "Образование",
	}
	result := make(map[string]string, len(names))
	for slug, name := range names {
		id := seedID("category:" + slug)
		if _, err := tx.Exec(ctx, `
			INSERT INTO categories (id, slug, name, is_default, created_at, updated_at)
			VALUES ($1, $2, $3, true, $4, $4)
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		`, id, slug, name, now); err != nil {
			return nil, fmt.Errorf("upsert demo category %s: %w", slug, err)
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1`, slug).Scan(&id); err != nil {
			return nil, fmt.Errorf("read demo category %s: %w", slug, err)
		}
		result[slug] = id
	}
	return result, nil
}

func buildDataset(now time.Time, categories map[string]string, ruleID string) dataset {
	now = dateOnly(now)
	historyStart := now.AddDate(-6, 0, 0)
	startMonth := time.Date(now.Year(), now.Month(), 1, 9, 0, 0, 0, now.Location()).AddDate(-6, 0, 0)
	checking, card := seedID("account:checking"), seedID("account:card")
	savings, deposit, usd := seedID("account:savings"), seedID("account:deposit"), seedID("account:usd")
	data := dataset{balances: map[string]decimal.Decimal{}}
	add := func(key, accountID, kind, amount, category, description string, occurred time.Time, related *string) string {
		id := seedID("transaction:" + key)
		var categoryID *string
		if category != "" {
			value := categories[category]
			categoryID = &value
		}
		data.transactions = append(data.transactions, transactionSeed{id: id, accountID: accountID, relatedAccountID: related, transactionType: kind, amount: amount, categoryID: categoryID, description: description, occurredAt: occurred, createdAt: occurred.Add(2 * time.Hour)})
		value := decimal.RequireFromString(amount)
		if kind == "expense" || kind == "transfer_out" {
			value = value.Neg()
		}
		data.balances[accountID] = data.balances[accountID].Add(value)
		return id
	}
	initial := []struct{ account, amount string }{{checking, "150000"}, {card, "100000"}, {savings, "100000"}, {deposit, "300000"}, {usd, "1500"}}
	for i, item := range initial {
		add(fmt.Sprintf("initial:%d", i), item.account, "initial_balance", item.amount, "", "Начальный баланс", historyStart, nil)
	}

	depositBalance := decimal.RequireFromString("300000")
	for monthIndex := 0; monthIndex <= 72; monthIndex++ {
		month := startMonth.AddDate(0, monthIndex, 0)
		yearIndex := monthIndex / 12
		addIfPast := func(key, account, kind, amount, category, description string, day int) string {
			date := time.Date(month.Year(), month.Month(), day, 9, 0, 0, 0, now.Location())
			if date.Before(historyStart) || date.After(now) {
				return ""
			}
			return add(fmt.Sprintf("%s:%02d", key, monthIndex), account, kind, amount, category, description, date, nil)
		}
		salary := 165000 + yearIndex*14000
		rent := 42000 + yearIndex*3500
		utilities := 6200 + (int(month.Month())%4)*650
		addIfPast("salary", checking, "income", fmt.Sprintf("%d", salary), "salary", "Зарплата", 1)
		addIfPast("rent", card, "expense", fmt.Sprintf("%d", rent), "housing", "Аренда квартиры", 4)
		addIfPast("utilities", card, "expense", fmt.Sprintf("%d", utilities), "housing", "Коммунальные услуги", 7)
		if month.Year() != now.Year() || month.Month() != now.Month() {
			addIfPast("spotify", card, "expense", fmt.Sprintf("%d", 650+yearIndex*90), "subscriptions", "Музыка", 8)
			addIfPast("vps", card, "expense", fmt.Sprintf("%d", 990+yearIndex*120), "subscriptions", "Облачный сервер", 18)
		}
		addIfPast("entertainment", card, "expense", fmt.Sprintf("%d", 3500+(monthIndex%5)*900), "entertainment", "Кино и развлечения", 20)
		interestAmount := 2800 + yearIndex*220
		interestID := addIfPast("interest", deposit, "interest_income", fmt.Sprintf("%d", interestAmount), "deposit_interest", "Проценты по вкладу", 25)
		if interestID != "" {
			depositBalance = depositBalance.Add(decimal.NewFromInt(int64(interestAmount)))
			date := time.Date(month.Year(), month.Month(), 25, 9, 0, 0, 0, now.Location())
			data.accruals = append(data.accruals, accrualSeed{seedID(fmt.Sprintf("accrual:%02d", monthIndex)), deposit, ruleID, interestID, fmt.Sprintf("%d", interestAmount), depositBalance.String(), date, date.Add(2 * time.Hour)})
		}
		addTransfer := func(key, from, to, amount string, day int) {
			date := time.Date(month.Year(), month.Month(), day, 10, 0, 0, 0, now.Location())
			if date.Before(historyStart) || date.After(now) {
				return
			}
			fromCopy, toCopy := to, from
			transferID := seedID(fmt.Sprintf("transfer:%s:%02d", key, monthIndex))
			outID := add(fmt.Sprintf("%s:out:%02d", key, monthIndex), from, "transfer_out", amount, "", "Перевод", date, &fromCopy)
			data.transactions[len(data.transactions)-1].transferID = &transferID
			inID := add(fmt.Sprintf("%s:in:%02d", key, monthIndex), to, "transfer_in", amount, "", "Перевод", date, &toCopy)
			data.transactions[len(data.transactions)-1].transferID = &transferID
			data.transfers = append(data.transfers, transferSeed{transferID, from, to, outID, inID, amount, "RUB", date.Add(time.Hour)})
		}
		addTransfer("reserve", checking, savings, "30000", 2)
		addTransfer("card", checking, card, "80000", 3)
		if monthIndex%12 == 0 {
			addIfPast("adjustment", checking, "adjustment", "2500", "other", "Корректировка остатка", 15)
		}
		if monthIndex%3 == 0 {
			addIfPast("health", card, "expense", fmt.Sprintf("%d", 2800+(monthIndex%4)*1100), "health", "Аптека и медицина", 12)
		}
		if month.Month() == time.June {
			addIfPast("travel", card, "expense", fmt.Sprintf("%d", 65000+yearIndex*12000), "travel", "Летняя поездка", 14)
		}
		if month.Month() == time.September {
			addIfPast("education", card, "expense", fmt.Sprintf("%d", 18000+yearIndex*2500), "education", "Курсы и книги", 10)
		}
		if month.Month() == time.December {
			addIfPast("gifts", card, "expense", fmt.Sprintf("%d", 22000+yearIndex*3000), "gifts", "Новогодние подарки", 22)
		}
		if monthIndex%18 == 6 {
			addIfPast("clothing", card, "expense", fmt.Sprintf("%d", 14000+(monthIndex%3)*3500), "clothing", "Одежда и обувь", 16)
		}
	}

	foodDescriptions := []string{"Пятёрочка", "ВкусВилл", "Перекрёсток", "Лента", "Рынок"}
	transportDescriptions := []string{"Метро и автобус", "Такси", "Каршеринг", "Топливо"}
	weekIndex := 0
	for day := historyStart; !day.After(now); day = day.AddDate(0, 0, 7) {
		if day.Year() == now.Year() && day.Month() == now.Month() {
			continue
		}
		foodAmount := 2600 + (weekIndex%7)*310
		add(fmt.Sprintf("food:%03d", weekIndex), card, "expense", fmt.Sprintf("%d", foodAmount), "food", foodDescriptions[weekIndex%len(foodDescriptions)], day, nil)
		if weekIndex%2 == 0 {
			transportAmount := 700 + (weekIndex%5)*240
			add(fmt.Sprintf("transport:%03d", weekIndex), card, "expense", fmt.Sprintf("%d", transportAmount), "transport", transportDescriptions[weekIndex%len(transportDescriptions)], day.AddDate(0, 0, 2), nil)
		}
		weekIndex++
	}
	currentMonth := time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, now.Location())
	add("current:food", card, "expense", "45000", "food", "Продукты за текущий месяц", safePastDate(currentMonth, now, 3), nil)
	add("current:transport", card, "expense", "8300", "transport", "Транспорт за текущий месяц", safePastDate(currentMonth, now, 5), nil)
	add("current:subscriptions", card, "expense", "11000", "subscriptions", "Подписки за текущий месяц", safePastDate(currentMonth, now, 8), nil)
	return data
}

func insertTransactions(ctx context.Context, tx pgx.Tx, transactions []transactionSeed) error {
	rows := make([][]any, 0, len(transactions))
	for i := range transactions {
		item := &transactions[i]
		rows = append(rows, []any{item.id, item.accountID, item.relatedAccountID, item.transferID, item.transactionType, item.amount, item.categoryID, item.description, item.occurredAt, item.createdAt})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"transactions"}, []string{"id", "account_id", "related_account_id", "transfer_id", "type", "amount", "category_id", "description", "occurred_at", "created_at"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("copy demo transactions: %w", err)
	}
	return nil
}

func insertTransfers(ctx context.Context, tx pgx.Tx, userID string, transfers []transferSeed) error {
	for i := range transfers {
		item := &transfers[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO transfers
				(id, user_id, from_account_id, to_account_id, from_transaction_id, to_transaction_id,
				 from_amount, to_amount, from_currency, to_currency, exchange_rate, exchange_rate_scale,
				 exchange_rate_provider, exchange_rate_date, fee_amount, status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$7,$8,$8,1,18,'demo',$9,0,'completed',$9,$9)
		`, item.id, userID, item.fromAccountID, item.toAccountID, item.fromTransactionID, item.toTransactionID, item.amount, item.currency, item.createdAt); err != nil {
			return fmt.Errorf("insert demo transfer: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE transactions SET transfer_id = $1 WHERE id IN ($2, $3)`, item.id, item.fromTransactionID, item.toTransactionID); err != nil {
			return fmt.Errorf("link demo transfer transactions: %w", err)
		}
	}
	return nil
}

func insertAccruals(ctx context.Context, tx pgx.Tx, accruals []accrualSeed) error {
	for i := range accruals {
		item := &accruals[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO interest_accruals (id, account_id, rule_id, transaction_id, accrual_date, amount, balance, annual_rate_bps, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,1200,$8)
		`, item.id, item.accountID, item.ruleID, item.transactionID, item.date, item.amount, item.balance, item.createdAt); err != nil {
			return fmt.Errorf("insert demo accrual: %w", err)
		}
	}
	return nil
}

func insertGoalsAndLimits(ctx context.Context, tx pgx.Tx, userID string, now time.Time, balances map[string]decimal.Decimal, categories map[string]string) error {
	type goalSeed struct {
		key, account, name, target, currency, status string
		date                                         *time.Time
	}
	future := dateOnly(now).AddDate(1, 0, 0)
	goals := []goalSeed{
		{"reserve", seedID("account:savings"), "Финансовая подушка", targetForRatio(balances[seedID("account:savings")], 0.70), "RUB", "active", &future},
		{"deposit", seedID("account:deposit"), "Цель по вкладу", balances[seedID("account:deposit")].StringFixed(2), "RUB", "active", nil},
		{"travel", seedID("account:usd"), "Путешествие", targetForRatio(balances[seedID("account:usd")], 0.35), "USD", "active", &future},
		{"inactive", seedID("account:card"), "Архивная цель", "500000.00", "RUB", "archived", nil},
	}
	for _, goal := range goals {
		if _, err := tx.Exec(ctx, `
			INSERT INTO financial_goals (id, owner_user_id, account_id, name, target_amount, currency, target_date, status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
		`, seedID("goal:"+goal.key), userID, goal.account, goal.name, goal.target, goal.currency, goal.date, goal.status, dateOnly(now)); err != nil {
			return fmt.Errorf("insert demo goal: %w", err)
		}
	}
	limits := []struct {
		key, slug, amount string
		active            bool
	}{
		{"food", "food", "100000", true},
		{"transport", "transport", "10000", true},
		{"subscriptions", "subscriptions", "10000", true},
		{"housing", "housing", "80000", false},
	}
	for _, limit := range limits {
		if _, err := tx.Exec(ctx, `
			INSERT INTO category_limits (id, owner_user_id, category_id, amount, currency, is_active, created_at, updated_at)
			VALUES ($1,$2,$3,$4,'RUB',$5,$6,$6)
		`, seedID("limit:"+limit.key), userID, categories[limit.slug], limit.amount, limit.active, dateOnly(now)); err != nil {
			return fmt.Errorf("insert demo category limit: %w", err)
		}
	}
	return nil
}

func targetForRatio(balance decimal.Decimal, ratio float64) string {
	return balance.Div(decimal.NewFromFloat(ratio)).Round(2).StringFixed(2)
}

func seedID(key string) string { return uuid.NewSHA1(namespace, []byte(key)).String() }
func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func safePastDate(monthStart, now time.Time, day int) time.Time {
	value := time.Date(monthStart.Year(), monthStart.Month(), day, 12, 0, 0, 0, now.Location())
	if value.After(now) {
		return dateOnly(now)
	}
	return value
}
