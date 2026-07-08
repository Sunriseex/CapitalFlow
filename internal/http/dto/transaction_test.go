package dto

import (
	"encoding/json"
	"testing"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestTransactionFromModelIncludesSource(t *testing.T) {
	refID := "11111111-1111-1111-1111-111111111111"
	response := TransactionFromModel(&models.Transaction{
		SourceType:     models.TransactionSourceCSVImport,
		SourceRefID:    &refID,
		SourceMetadata: json.RawMessage(`{"parser_version":"1"}`),
	})

	if response.SourceType != models.TransactionSourceCSVImport {
		t.Fatalf("source type = %q, want csv_import", response.SourceType)
	}
	if response.SourceRefID == nil || *response.SourceRefID != refID {
		t.Fatalf("source ref = %v, want %q", response.SourceRefID, refID)
	}
	if string(response.SourceMetadata) != `{"parser_version":"1"}` {
		t.Fatalf("source metadata = %s", response.SourceMetadata)
	}
}

func TestTransactionFromModelDefaultsManualSource(t *testing.T) {
	response := TransactionFromModel(&models.Transaction{})
	if response.SourceType != models.TransactionSourceManual {
		t.Fatalf("source type = %q, want manual", response.SourceType)
	}
	if string(response.SourceMetadata) != `{}` {
		t.Fatalf("source metadata = %s, want empty object", response.SourceMetadata)
	}
}
