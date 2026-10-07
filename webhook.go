package assinafy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
)

// ErrInvalidWebhookSignature reports a delivery whose Standard Webhooks headers
// are missing, stale or do not match the body. Reject such a request with a
// non-2xx status.
var ErrInvalidWebhookSignature = errors.New("assinafy: invalid webhook signature")

// WebhookTolerance is the largest difference between the webhook-timestamp
// header and the local clock that VerifyRequest accepts, limiting replays.
const WebhookTolerance = 5 * time.Minute

// WebhookVerifier checks webhook signatures and decodes payloads. The zero
// value is not usable; call NewWebhookVerifier.
type WebhookVerifier struct {
	secret []byte
}

// NewWebhookVerifier creates a WebhookVerifier bound to an endpoint's signing
// secret, as returned by WebhookResource.GetEndpointSecret ("whsec_" followed by
// the base64 key).
func NewWebhookVerifier(secret string) *WebhookVerifier {
	return &WebhookVerifier{secret: []byte(secret)}
}

// VerifyRequest checks a delivery to an endpoint with SigningEnabled, following
// the Standard Webhooks specification. Pass the request headers and the raw body
// exactly as received. It returns nil when one v1 entry of webhook-signature
// equals base64(HMAC-SHA256(key, "{webhook-id}.{webhook-timestamp}.{body}")) and
// webhook-timestamp is within WebhookTolerance of the local clock. A malformed
// secret wraps errors.ErrInvalidInput; any other failure wraps
// ErrInvalidWebhookSignature. Deduplicate verified deliveries by webhook-id.
func (v *WebhookVerifier) VerifyRequest(header http.Header, body []byte) error {
	return v.verifyAt(header, body, time.Now())
}

func (v *WebhookVerifier) verifyAt(header http.Header, body []byte, now time.Time) error {
	if v == nil {
		return fmt.Errorf("%w: nil WebhookVerifier", sdkerrors.ErrInvalidInput)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(string(v.secret), "whsec_"))
	if err != nil || len(key) == 0 {
		return fmt.Errorf("%w: webhook secret must be whsec_ followed by a base64 key", sdkerrors.ErrInvalidInput)
	}
	id, timestamp := header.Get("webhook-id"), header.Get("webhook-timestamp")
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if id == "" || err != nil {
		return fmt.Errorf("%w: missing webhook-id or webhook-timestamp", ErrInvalidWebhookSignature)
	}
	if age := now.Sub(time.Unix(seconds, 0)); age > WebhookTolerance || age < -WebhookTolerance {
		return fmt.Errorf("%w: webhook-timestamp outside tolerance", ErrInvalidWebhookSignature)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "."))
	mac.Write(body)
	expected := mac.Sum(nil)
	for _, entry := range strings.Fields(header.Get("webhook-signature")) {
		version, signature, _ := strings.Cut(entry, ",")
		got, err := base64.StdEncoding.DecodeString(signature)
		if version == "v1" && err == nil && hmac.Equal(got, expected) {
			return nil
		}
	}
	return ErrInvalidWebhookSignature
}

// Verify performs a constant-time check of the hex-encoded HMAC-SHA256 of
// payload, keyed with the raw secret string, against the supplied signature.
//
// Deprecated: Assinafy signs deliveries with Standard Webhooks; use
// VerifyRequest.
func (v *WebhookVerifier) Verify(payload []byte, signature string) bool {
	if v == nil || len(v.secret) == 0 || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ExtractEvent decodes a webhook payload into a typed WebhookPayload. Call it
// after VerifyRequest succeeds.
func (v *WebhookVerifier) ExtractEvent(payload []byte) (*models.WebhookPayload, error) {
	var event models.WebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("assinafy: parse webhook payload: %w", err)
	}
	return &event, nil
}
