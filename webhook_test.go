package assinafy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
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

	t.Run("empty secret", func(t *testing.T) {
		empty := NewWebhookVerifier("")
		if empty.Verify(payload, sign(t, "", payload)) {
			t.Error("empty secret must not verify payloads")
		}
	})

	t.Run("nil verifier", func(t *testing.T) {
		var nilVerifier *WebhookVerifier
		if nilVerifier.Verify(payload, "signature") {
			t.Error("nil verifier must not verify payloads")
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

// The Standard Webhooks specification's published test vector.
const (
	standardSecret    = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw" // #nosec G101 -- specification fixture
	standardID        = "msg_p5jXN8AQM9LWM0D4loKWxJek"
	standardTimestamp = "1614265330"
	standardSignature = "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE="
	standardBody      = `{"test": 2432232314}`
)

func standardHeaders(signature string) http.Header {
	h := http.Header{}
	h.Set("webhook-id", standardID)
	h.Set("webhook-timestamp", standardTimestamp)
	h.Set("webhook-signature", signature)
	return h
}

func TestWebhookVerifierVerifyRequest(t *testing.T) {
	sent := time.Unix(1614265330, 0)
	v := NewWebhookVerifier(standardSecret)

	for _, tc := range []struct {
		name     string
		verifier *WebhookVerifier
		header   http.Header
		body     string
		now      time.Time
		want     error
	}{
		{"valid", v, standardHeaders(standardSignature), standardBody, sent, nil},
		{"one of several entries", v, standardHeaders("v1,AAAA " + standardSignature), standardBody, sent, nil},
		{"secret without prefix", NewWebhookVerifier(standardSecret[len("whsec_"):]), standardHeaders(standardSignature), standardBody, sent, nil},
		{"within tolerance", v, standardHeaders(standardSignature), standardBody, sent.Add(WebhookTolerance), nil},
		{"tampered body", v, standardHeaders(standardSignature), `{"test": 1}`, sent, ErrInvalidWebhookSignature},
		{"other version", v, standardHeaders("v2" + standardSignature[2:]), standardBody, sent, ErrInvalidWebhookSignature},
		{"missing signature", v, standardHeaders(""), standardBody, sent, ErrInvalidWebhookSignature},
		{"stale", v, standardHeaders(standardSignature), standardBody, sent.Add(WebhookTolerance + time.Second), ErrInvalidWebhookSignature},
		{"future", v, standardHeaders(standardSignature), standardBody, sent.Add(-WebhookTolerance - time.Second), ErrInvalidWebhookSignature},
		{"missing id", v, func() http.Header { h := standardHeaders(standardSignature); h.Del("webhook-id"); return h }(), standardBody, sent, ErrInvalidWebhookSignature},
		{"bad timestamp", v, func() http.Header { h := standardHeaders(standardSignature); h.Set("webhook-timestamp", "x"); return h }(), standardBody, sent, ErrInvalidWebhookSignature},
		{"malformed secret", NewWebhookVerifier("whsec_%%"), standardHeaders(standardSignature), standardBody, sent, sdkerrors.ErrInvalidInput},
		{"empty secret", NewWebhookVerifier(""), standardHeaders(standardSignature), standardBody, sent, sdkerrors.ErrInvalidInput},
		{"nil verifier", nil, standardHeaders(standardSignature), standardBody, sent, sdkerrors.ErrInvalidInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.verifier.verifyAt(tc.header, []byte(tc.body), tc.now)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}

	if err := v.VerifyRequest(standardHeaders(standardSignature), []byte(standardBody)); !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("VerifyRequest with a 2021 timestamp = %v, want stale rejection", err)
	}
}
