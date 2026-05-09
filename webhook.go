package assinafy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/assinafy/golang-sdk/models"
)

// WebhookVerifier validates webhook signatures and decodes payloads. The zero
// value is not usable; call NewWebhookVerifier.
type WebhookVerifier struct {
	secret []byte
}

// NewWebhookVerifier creates a WebhookVerifier bound to the given shared secret.
func NewWebhookVerifier(secret string) *WebhookVerifier {
	return &WebhookVerifier{secret: []byte(secret)}
}

// Verify performs a constant-time comparison of the hex-encoded HMAC-SHA256
// of payload against the supplied signature.
func (v *WebhookVerifier) Verify(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ExtractEvent decodes a webhook payload into a typed WebhookPayload.
func (v *WebhookVerifier) ExtractEvent(payload []byte) (*models.WebhookPayload, error) {
	var event models.WebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("assinafy: parse webhook payload: %w", err)
	}
	return &event, nil
}
