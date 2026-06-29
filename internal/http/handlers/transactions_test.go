package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestApplyTransactionListFilter(t *testing.T) {
	categoryID := "11111111-1111-1111-1111-111111111111"
	transactions := []models.Transaction{
		{
			ID:          "old-income",
			Type:        models.TransactionTypeIncome,
			Amount:      dec("100"),
			CategoryID:  &categoryID,
			Description: "Salary May",
			OccurredAt:  time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          "expense",
			Type:        models.TransactionTypeExpense,
			Amount:      dec("30"),
			Description: "Food",
			OccurredAt:  time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          "new-income",
			Type:        models.TransactionTypeIncome,
			Amount:      dec("200"),
			CategoryID:  &categoryID,
			Description: "Salary June",
			OccurredAt:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	got := applyTransactionListFilter(transactions, &repository.TransactionListFilter{
		CategoryID: categoryID,
		Type:       models.TransactionTypeIncome,
		FromDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:     time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		Search:     "salary",
		Limit:      1,
		Page:       2,
	})

	if len(got) != 1 {
		t.Fatalf("filtered len = %d, want 1", len(got))
	}
	if got[0].ID != "old-income" {
		t.Fatalf("filtered transaction = %s, want old-income", got[0].ID)
	}
}

func TestApplyTransactionListFilterUsesStableNewestFirstOrder(t *testing.T) {
	stamp := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	transactions := []models.Transaction{
		{ID: "00000000-0000-0000-0000-000000000001", OccurredAt: stamp, CreatedAt: stamp},
		{ID: "00000000-0000-0000-0000-000000000003", OccurredAt: stamp, CreatedAt: stamp},
		{ID: "00000000-0000-0000-0000-000000000002", OccurredAt: stamp, CreatedAt: stamp},
	}

	for page, want := range []string{
		"00000000-0000-0000-0000-000000000003",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000001",
	} {
		got := applyTransactionListFilter(transactions, &repository.TransactionListFilter{Limit: 1, Page: page + 1})
		if len(got) != 1 || got[0].ID != want {
			t.Fatalf("page %d = %#v, want %s", page+1, got, want)
		}
	}
}

func TestParseTransactionListFilterRejectsInvalidQuery(t *testing.T) {
	tests := []string{
		"/api/v1/transactions?account_id=bad",
		"/api/v1/transactions?category_id=bad",
		"/api/v1/transactions?type=bad",
		"/api/v1/transactions?from_date=2026-13-01",
		"/api/v1/transactions?from_date=2026-06-01&to_date=2026-05-01",
		"/api/v1/transactions?limit=0",
		"/api/v1/transactions?page=-1",
	}

	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
			rec := httptest.NewRecorder()

			_, ok := parseTransactionListFilter(rec, req)

			if ok {
				t.Fatal("filter parse succeeded, want rejection")
			}
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestListTransactionsUsesRepositoryFiltering(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	categoryID := "22222222-2222-2222-2222-222222222222"
	transactions := &testTransactionRepo{
		listFilteredTransactions: []models.Transaction{
			{
				ID:          "33333333-3333-3333-3333-333333333333",
				AccountID:   "11111111-1111-1111-1111-111111111111",
				Type:        models.TransactionTypeIncome,
				Amount:      dec("1"),
				CategoryID:  &categoryID,
				Description: "Salary June",
				OccurredAt:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
				CreatedAt:   time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	store.transactions = transactions
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := newTestRouter(store, &RouterConfig{}, tokens)
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/transactions?account_id=11111111-1111-1111-1111-111111111111&category_id="+categoryID+"&type=income&from_date=2026-05-01&to_date=2026-06-30&search=salary&limit=10&page=2",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if transactions.listFilteredCalls != 1 {
		t.Fatalf("ListByUserFiltered calls = %d, want 1", transactions.listFilteredCalls)
	}
	if transactions.listFilteredUserID != "user-1" {
		t.Fatalf("filtered user id = %q, want user-1", transactions.listFilteredUserID)
	}
	filter := transactions.listFilteredFilter
	if filter.AccountID != "11111111-1111-1111-1111-111111111111" ||
		filter.CategoryID != categoryID ||
		filter.Type != models.TransactionTypeIncome ||
		filter.Search != "salary" ||
		filter.Limit != 10 ||
		filter.Page != 2 ||
		filter.FromDate.IsZero() ||
		filter.ToDate.IsZero() {
		t.Fatalf("unexpected filter: %+v", filter)
	}
}

func TestCreateTransactionRejectsTransferTypes(t *testing.T) {
	tests := []models.TransactionType{
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
	}

	for _, transactionType := range tests {
		t.Run(string(transactionType), func(t *testing.T) {
			tokens, pair := testProfileTokenPair(t)
			store := newTestProfileStore()
			store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

			router := newTestRouter(store, &RouterConfig{}, tokens)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transactions", strings.NewReader(`{
				"account_id":"11111111-1111-1111-1111-111111111111",
				"type":"`+string(transactionType)+`",
				"amount":"100"
			}`))
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			req.Header.Set("Idempotency-Key", "reject-direct-"+string(transactionType))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestCreateTransactionUsesUserScopedCreate(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	transactions := &testTransactionRepo{}
	store.transactions = transactions
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := newTestRouter(store, &RouterConfig{}, tokens)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transactions", strings.NewReader(`{
		"account_id":"11111111-1111-1111-1111-111111111111",
		"type":"income",
		"amount":"100"
	}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "create-transaction-user-scoped")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if transactions.oldCreateCalls != 0 {
		t.Fatalf("old Create calls = %d, want 0", transactions.oldCreateCalls)
	}
	if transactions.createForUserCalls != 1 {
		t.Fatalf("CreateForUser calls = %d, want 1", transactions.createForUserCalls)
	}
}

func TestCreateTransactionForOtherUsersAccountReturnsNotFound(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.transactions = &testTransactionRepo{createForUserErr: repository.ErrNotFound}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := newTestRouter(store, &RouterConfig{}, tokens)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transactions", strings.NewReader(`{
		"account_id":"22222222-2222-2222-2222-222222222222",
		"type":"income",
		"amount":"100"
	}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "create-transaction-other-account")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestDeleteTransactionRouteIsRemoved(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := newTestRouter(store, &RouterConfig{}, tokens)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/transactions/33333333-3333-3333-3333-333333333333", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
}

func TestRecalculateInterestRejectsInvalidRequestBeforeStoreAccess(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		body      string
	}{
		{
			name:      "invalid account id",
			accountID: "not-a-uuid",
			body:      `{}`,
		},
		{
			name:      "invalid body",
			accountID: "11111111-1111-1111-1111-111111111111",
			body:      `{`,
		},
		{
			name:      "invalid rule id",
			accountID: "11111111-1111-1111-1111-111111111111",
			body:      `{"rule_id":"not-a-uuid"}`,
		},
		{
			name:      "invalid date",
			accountID: "11111111-1111-1111-1111-111111111111",
			body:      `{"from_date":"2026-13-01"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				"/api/v1/accounts/"+tt.accountID+"/recalculate-interest",
				strings.NewReader(tt.body),
			)
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("id", tt.accountID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
			rec := httptest.NewRecorder()

			new(Handler).recalculateInterest(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}
