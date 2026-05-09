package assinafy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sign(t *testing.T, secret string, payload []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestWebhookVerifierVerify(t *testing.T) {
	secret := "test-webhook-secret" // #nosec G101 -- fixture, not a real credential
	v := NewWebhookVerifier(secret)
	payload := []byte(`{"event":"document_ready","payload":{}}`)

	t.Run("valid signature", func(t *testing.T) {
		if !v.Verify(payload, sign(t, secret, payload)) {
			t.Error("expected valid signature")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		if v.Verify(payload, "deadbeef") {
			t.Error("expected invalid signature")
		}
	})

	t.Run("tampered payload", func(t *testing.T) {
		signature := sign(t, secret, payload)
		if v.Verify([]byte(`{"event":"document_deleted"}`), signature) {
			t.Error("expected verification to fail for tampered payload")
		}
	})
}

func TestWebhookVerifierExtractEvent(t *testing.T) {
	v := NewWebhookVerifier("secret")

	t.Run("valid payload", func(t *testing.T) {
		payload := []byte(`{"id":1,"event":"document_ready","created_at":1705316400,"payload":{"document_id":"123"},"account_id":"acc_123"}`)
		event, err := v.ExtractEvent(payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Event != "document_ready" {
			t.Errorf("got event %q, want document_ready", event.Event)
		}
		if event.CreatedAt.String() != "1705316400" {
			t.Errorf("got created_at %q, want 1705316400", event.CreatedAt)
		}
		if event.Payload["document_id"] != "123" {
			t.Errorf("got payload %v, want document_id=123", event.Payload)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		if _, err := v.ExtractEvent([]byte("not json")); err == nil {
			t.Error("expected error")
		}
	})
}
