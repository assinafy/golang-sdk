package models

import (
	"encoding/json"
	"testing"
)

func TestUpdateEnvelopeRequestMessage(t *testing.T) {
	for _, tc := range []struct {
		body UpdateEnvelopeRequest
		want string
	}{
		{UpdateEnvelopeRequest{}, `{}`},
		{UpdateEnvelopeRequest{Message: &Nullable[string]{}}, `{"message":null}`},
		{UpdateEnvelopeRequest{Message: NullableValue("")}, `{"message":""}`},
		{UpdateEnvelopeRequest{DocumentIDs: []string{}}, `{"document_ids":[]}`},
	} {
		got, err := json.Marshal(tc.body)
		if err != nil || string(got) != tc.want {
			t.Errorf("Marshal = %s, %v; want %s", got, err, tc.want)
		}
	}
}

func TestEnvelopeResponseNulls(t *testing.T) {
	var envelope DocumentEnvelope
	if err := json.Unmarshal([]byte(`{"id":"env","message":null,"documents":[],"counts":null,"signing_urls":[],"created_at":"2026-11-01T12:00:00Z"}`), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Counts != nil || envelope.Message != nil || len(envelope.Documents) != 0 {
		t.Fatalf("envelope = %+v", envelope)
	}
	var signer EnvelopeDocumentSigner
	if err := json.Unmarshal([]byte(`{"id":"s","full_name":"Signer","email":null,"whatsapp_phone_number":null,"completed":true,"signed_at":"2026-11-02T12:00:00Z"}`), &signer); err != nil {
		t.Fatal(err)
	}
	if !signer.Completed || signer.SignedAt == nil || signer.SignedAt.String() != "2026-11-02T12:00:00Z" {
		t.Fatalf("signer = %+v", signer)
	}
}
