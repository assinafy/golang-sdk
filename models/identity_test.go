package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestIdentityReviewPayload(t *testing.T) {
	const payload = `{
		"id":"verification", "status":"InReview", "provider":"AssinafyVerify",
		"verification_method":"IdDocumentFaceMatchBasic", "evidence_available":true,
		"reason_codes":[{"code":"NAME_MISMATCH","description":"Name requires review"}],
		"expected":{"name":"Example Signer","cpf":null},
		"document":{"type":"CNH","name":"Example Signer","cpf":null},
		"matches":{"cpf":false,"name":0,"birth_date":null},
		"checks":[{"name":"name","result":"review","score":0,"reasons":["NAME_MISMATCH"]}],
		"images":["document_front","document_back","document_pdf","face"],
		"ruleset_version":"rules-1", "evidence_sha256":"digest",
		"reviewed_by":{"id":"reviewer","name":"Example Reviewer"},
		"reviewed_at":"2026-10-16T12:00:00Z", "review_note":"Identity confirmed",
		"completed_at":null
	}`
	var review SignerIdentityReview
	if err := json.Unmarshal([]byte(payload), &review); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(payload), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("review payload lost fields: %s", encoded)
	}
	if err := json.Unmarshal([]byte(`{"expected":null,"document":null,"matches":null,"reviewed_by":null}`), &review); err != nil {
		t.Fatal(err)
	}
	if review.Expected != nil || review.Document != nil || review.Matches != nil || review.ReviewedBy != nil {
		t.Error("nullable evidence retained stale values")
	}
}

func TestAssignmentIdentityVerificationStatus(t *testing.T) {
	var doc Document
	if err := json.Unmarshal([]byte(`{"status":"pending_identity_review","assignment":{"signers":[{"id":"signer","identity_verification_status":"InReview"}]}}`), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Status != StatusPendingIdentityReview || doc.Assignment == nil || len(doc.Assignment.Signers) != 1 {
		t.Fatalf("document = %+v", doc)
	}
	if status := doc.Assignment.Signers[0].IdentityVerificationStatus; status == nil || *status != "InReview" {
		t.Errorf("identity verification status = %v", status)
	}
}
