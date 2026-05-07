package assinafy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/assinafy/assinafy-go/models"
)

func TestWebhookVerifier(t *testing.T) {
	secret := "test-webhook-secret"
	verifier := NewWebhookVerifier(secret)

	t.Run("valid signature", func(t *testing.T) {
		payload := []byte(`{"event":"document_ready","data":{}}`)

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))

		if !verifier.Verify(payload, signature) {
			t.Error("expected signature to be valid")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		payload := []byte(`{"event":"document_ready","data":{}}`)
		wrongSignature := "invalid-signature"

		if verifier.Verify(payload, wrongSignature) {
			t.Error("expected signature to be invalid")
		}
	})

	t.Run("tampered payload", func(t *testing.T) {
		payload := []byte(`{"event":"document_ready","data":{}}`)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))

		tamperedPayload := []byte(`{"event":"document_deleted","data":{}}`)
		if verifier.Verify(tamperedPayload, signature) {
			t.Error("expected tampered payload to fail verification")
		}
	})
}

func TestWebhookVerifier_ExtractEvent(t *testing.T) {
	verifier := NewWebhookVerifier("secret")

	t.Run("valid payload", func(t *testing.T) {
		payload := []byte(`{"event":"document_ready","timestamp":"2024-01-01T00:00:00Z","data":{"document_id":"123"}}`)
		event, err := verifier.ExtractEvent(payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Event != "document_ready" {
			t.Errorf("expected 'document_ready' but got '%s'", event.Event)
		}
		if event.Timestamp != "2024-01-01T00:00:00Z" {
			t.Errorf("unexpected timestamp: %s", event.Timestamp)
		}
	})

	t.Run("invalid payload", func(t *testing.T) {
		payload := []byte(`invalid json`)
		_, err := verifier.ExtractEvent(payload)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestWebhookVerifier_GetEventType(t *testing.T) {
	verifier := NewWebhookVerifier("secret")

	event := &models.WebhookPayload{
		Event: "signer_signed_document",
	}

	if verifier.GetEventType(event) != "signer_signed_document" {
		t.Errorf("unexpected event type: %s", verifier.GetEventType(event))
	}
}

func TestWebhookVerifier_GetEventData(t *testing.T) {
	verifier := NewWebhookVerifier("secret")

	expectedData := map[string]interface{}{
		"document_id": "123",
		"signer_id":   "456",
	}

	event := &models.WebhookPayload{
		Event: "document_ready",
		Data:  expectedData,
	}

	data := verifier.GetEventData(event)
	if data["document_id"] != "123" {
		t.Errorf("unexpected data: %v", data)
	}
	if data["signer_id"] != "456" {
		t.Errorf("unexpected data: %v", data)
	}
}
