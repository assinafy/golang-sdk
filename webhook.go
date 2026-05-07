package assinafy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/assinafy/assinafy-go/models"
)

type WebhookVerifier struct {
	secret []byte
}

func NewWebhookVerifier(secret string) *WebhookVerifier {
	return &WebhookVerifier{
		secret: []byte(secret),
	}
}

func (v *WebhookVerifier) Verify(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

func (v *WebhookVerifier) ExtractEvent(payload []byte) (*models.WebhookPayload, error) {
	var event models.WebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}
	return &event, nil
}

func (v *WebhookVerifier) GetEventType(event *models.WebhookPayload) string {
	return event.Event
}

func (v *WebhookVerifier) GetEventData(event *models.WebhookPayload) map[string]interface{} {
	return event.Data
}
