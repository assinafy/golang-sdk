package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnlimitedDocumentBalance(t *testing.T) {
	var estimate CostEstimate
	if err := json.Unmarshal([]byte(`{"document_balance":null,"credit_balance":null}`), &estimate); err != nil {
		t.Fatal(err)
	}
	if !estimate.DocumentBalanceUnlimited || !estimate.CreditBalanceUnlimited {
		t.Fatal("null balances lost")
	}
	encoded, err := json.Marshal(estimate)
	if err != nil || !strings.Contains(string(encoded), `"document_balance":null`) {
		t.Fatalf("encoded=%s err=%v", encoded, err)
	}
	if err := json.Unmarshal([]byte(`{"document_balance":0}`), &estimate); err != nil || estimate.DocumentBalanceUnlimited || estimate.DocumentBalance != 0 {
		t.Fatalf("zero balance: %+v err=%v", estimate, err)
	}
	if err := json.Unmarshal([]byte(`{"document_balance":"invalid"}`), &estimate); err == nil {
		t.Fatal("invalid balance accepted")
	}
}

func TestClosedSigningDocument(t *testing.T) {
	var doc Document
	if err := json.Unmarshal([]byte(`{"id":"d","is_signer_access_closed":true,"envelope":{"id":"e","name":"Agreement"},"certified_copy":{"available":true,"notification_method":"Email","resends_left":null}}`), &doc); err != nil {
		t.Fatal(err)
	}
	if (doc.IsSignerAccessClosed == nil || !*doc.IsSignerAccessClosed) || doc.Envelope.ID != "e" || !doc.CertifiedCopy.Available || doc.CertifiedCopy.ResendsLeft != nil {
		t.Fatalf("closed signing document: %+v", doc)
	}
	if err := json.Unmarshal([]byte(`{"is_signer_access_closed":false}`), &doc); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(doc)
	if err != nil || !strings.Contains(string(encoded), `"is_signer_access_closed":false`) {
		t.Fatalf("open signer state lost: %s, %v", encoded, err)
	}
}
