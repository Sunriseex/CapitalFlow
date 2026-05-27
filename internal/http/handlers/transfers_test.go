package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
)

const (
	testTransferFromAccountID = "11111111-1111-1111-1111-111111111111"
	testTransferToAccountID   = "22222222-2222-2222-2222-222222222222"
	testForeignAccountID      = "33333333-3333-3333-3333-333333333333"
)

func TestTransferRouteCreatesOwnedTransfer(t *testing.T) {
	router, transactions, token := newTestTransferRouter(t)
	req := newTestTransferRequest(t, token, "create-owned-transfer", `{
		"from_account_id":"`+testTransferFromAccountID+`",
		"to_account_id":"`+testTransferToAccountID+`",
		"amount":"12500",
		"description":"Move savings"
	}`)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if transactions.createTransferCalls != 1 {
		t.Fatalf("create transfer calls = %d, want 1", transactions.createTransferCalls)
	}
	if transactions.createTransferUserID != "user-1" {
		t.Fatalf("create transfer user id = %q, want user-1", transactions.createTransferUserID)
	}
	if transactions.createTransferIdempotencyKey != "create-owned-transfer" {
		t.Fatalf("create transfer idempotency key = %q, want create-owned-transfer", transactions.createTransferIdempotencyKey)
	}
	if len(transactions.createTransferTransactions) != 2 {
		t.Fatalf("created transactions = %d, want 2", len(transactions.createTransferTransactions))
	}

	var response dto.TransferResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Out.AccountID != testTransferFromAccountID || response.Out.Type != models.TransactionTypeTransferOut {
		t.Fatalf("out transaction = %+v", response.Out)
	}
	if response.In.AccountID != testTransferToAccountID || response.In.Type != models.TransactionTypeTransferIn {
		t.Fatalf("in transaction = %+v", response.In)
	}
	if !response.Out.Amount.Equal(dec("12500")) || !response.In.Amount.Equal(dec("12500")) {
		t.Fatalf("amounts = out %s in %s, want 12500", response.Out.Amount.Decimal, response.In.Amount.Decimal)
	}
}

func TestTransferRouteCreatesTransferWithFee(t *testing.T) {
	router, transactions, token := newTestTransferRouter(t)
	req := newTestTransferRequest(t, token, "create-fee-transfer", `{
		"from_account_id":"`+testTransferFromAccountID+`",
		"to_account_id":"`+testTransferToAccountID+`",
		"amount":"12500",
		"fee_amount":"12.50",
		"description":"Move savings"
	}`)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(transactions.createTransferTransactions) != 3 {
		t.Fatalf("created transactions = %d, want 3", len(transactions.createTransferTransactions))
	}
	if transactions.createTransferTransactions[2].Type != models.TransactionTypeExpense {
		t.Fatalf("fee transaction = %+v", transactions.createTransferTransactions[2])
	}

	var response dto.TransferResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Fee == nil || response.Fee.Type != models.TransactionTypeExpense {
		t.Fatalf("fee response = %+v", response.Fee)
	}
}

func TestTransferRouteListsTransferEvents(t *testing.T) {
	router, transactions, token := newTestTransferRouter(t)
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	transactions.listTransfersByUser = []models.Transfer{{
		ID:                   "44444444-4444-4444-4444-444444444444",
		UserID:               "user-1",
		FromAccountID:        testTransferFromAccountID,
		ToAccountID:          testTransferToAccountID,
		FromTransactionID:    "55555555-5555-5555-5555-555555555555",
		ToTransactionID:      "66666666-6666-6666-6666-666666666666",
		FromAmount:           dec("12500"),
		ToAmount:             dec("12500"),
		FromCurrency:         "RUB",
		ToCurrency:           "RUB",
		ExchangeRate:         "1",
		ExchangeRateScale:    18,
		ExchangeRateProvider: "internal",
		ExchangeRateDate:     now,
		Status:               "completed",
		CreatedAt:            now,
		UpdatedAt:            now,
	}}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/transfers", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response []dto.TransferEventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 || response[0].ID != "44444444-4444-4444-4444-444444444444" {
		t.Fatalf("response = %+v", response)
	}
}

func TestTransferRouteRejectsForeignAccount(t *testing.T) {
	router, transactions, token := newTestTransferRouter(t)
	req := newTestTransferRequest(t, token, "reject-foreign-transfer", `{
		"from_account_id":"`+testTransferFromAccountID+`",
		"to_account_id":"`+testForeignAccountID+`",
		"amount":"12500"
	}`)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if transactions.createTransferCalls != 0 {
		t.Fatalf("create transfer calls = %d, want 0", transactions.createTransferCalls)
	}
}

func TestTransferRouteRejectsMalformedBody(t *testing.T) {
	router, transactions, token := newTestTransferRouter(t)
	req := newTestTransferRequest(t, token, "reject-malformed-transfer", `{`)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if transactions.createTransferCalls != 0 {
		t.Fatalf("create transfer calls = %d, want 0", transactions.createTransferCalls)
	}
}

func TestTransferRouteRejectsValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid from account id",
			body: `{
				"from_account_id":"not-a-uuid",
				"to_account_id":"` + testTransferToAccountID + `",
				"amount":"12500"
			}`,
		},
		{
			name: "same accounts",
			body: `{
				"from_account_id":"` + testTransferFromAccountID + `",
				"to_account_id":"` + testTransferFromAccountID + `",
				"amount":"12500"
			}`,
		},
		{
			name: "non positive amount",
			body: `{
				"from_account_id":"` + testTransferFromAccountID + `",
				"to_account_id":"` + testTransferToAccountID + `",
				"amount":"0"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, transactions, token := newTestTransferRouter(t)
			req := newTestTransferRequest(t, token, "reject-validation-"+strings.ReplaceAll(tt.name, " ", "-"), tt.body)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if transactions.createTransferCalls != 0 {
				t.Fatalf("create transfer calls = %d, want 0", transactions.createTransferCalls)
			}
		})
	}
}

func TestTransferRouteIdempotentRetryReplaysResponse(t *testing.T) {
	router, transactions, token := newTestTransferRouter(t)
	body := `{
		"from_account_id":"` + testTransferFromAccountID + `",
		"to_account_id":"` + testTransferToAccountID + `",
		"amount":"12500",
		"description":"Move savings"
	}`

	var firstBody string
	for i := range 2 {
		req := newTestTransferRequest(t, token, "transfer-retry", body)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, want %d: %s", i+1, rec.Code, http.StatusCreated, rec.Body.String())
		}
		if i == 0 {
			firstBody = rec.Body.String()
			continue
		}
		if rec.Body.String() != firstBody {
			t.Fatalf("replayed response body changed: got %s want %s", rec.Body.String(), firstBody)
		}
	}
	if transactions.createTransferCalls != 1 {
		t.Fatalf("create transfer calls = %d, want 1", transactions.createTransferCalls)
	}
}

func newTestTransferRouter(t *testing.T) (http.Handler, *testTransactionRepo, string) {
	t.Helper()

	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		testTransferFromAccountID: testAccount(testTransferFromAccountID, "user-1", "RUB"),
		testTransferToAccountID:   testAccount(testTransferToAccountID, "user-1", "RUB"),
		testForeignAccountID:      testAccount(testForeignAccountID, "user-2", "RUB"),
	}}
	transactions := &testTransactionRepo{transactionCountByAccount: map[string]int64{}}
	store.transactions = transactions
	store.users.byID["user-1"] = &models.User{
		ID:              "user-1",
		Email:           "user@example.com",
		PrimaryCurrency: "RUB",
	}
	store.refresh.byID[pair.RefreshTokenID] = &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    "user-1",
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	return NewRouter(store, &RouterConfig{TokenService: tokens}), transactions, pair.AccessToken
}

func newTestTransferRequest(t *testing.T, token, idempotencyKey, body string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transfers", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", idempotencyKey)
	return req
}
