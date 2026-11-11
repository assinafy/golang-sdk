package assinafy

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

// TestIntegrationAccountFeatures reads workspace entitlements, usage and
// notification settings without changing any preferences.
func TestIntegrationAccountFeatures(t *testing.T) {
	c, accountID := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	account, err := c.Accounts.GetWithParams(ctx, "", &models.AccountGetParams{Expand: "signing_features"})
	if err != nil {
		t.Fatalf("Accounts.GetWithParams: %v", err)
	}
	if account.ID != accountID || account.SigningFeatures == nil {
		t.Fatal("account omitted the requested signing features")
	}
	usage, err := c.Accounts.Usage(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.Usage: %v", err)
	}
	if usage.Balances.Documents.Spent < 0 || usage.Balances.Credits.Spent < 0 {
		t.Fatal("usage returned a negative consumed allowance")
	}
	policy, err := c.Accounts.GetNotificationPolicy(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.GetNotificationPolicy: %v", err)
	}
	if _, ok := policy.Events[models.NotificationPreferenceDocumentCompleted]; !ok {
		t.Fatal("workspace policy omitted the completion rule")
	}
	preferences, err := c.Users.GetPushPreferences(ctx)
	if err != nil {
		t.Fatalf("Users.GetPushPreferences: %v", err)
	}
	if _, ok := preferences[models.NotificationPreferenceSignerViewed]; !ok {
		t.Fatal("push preferences omitted the first-view rule")
	}
}

// TestIntegrationEmbedLifecycle registers a temporary origin and exchanges a
// single-use session for a disposable embedded signer. It sends an email
// invitation and cancellation, requires no credits, and does not submit an OTP.
func TestIntegrationEmbedLifecycle(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_EMBED_TESTS") != "1" {
		t.Skip("set ASSINAFY_RUN_EMBED_TESTS=1 to run embedded signing (sends emails)")
	}
	email := strings.TrimSpace(os.Getenv("ASSINAFY_TEST_EMAIL_PRIMARY"))
	if email == "" {
		t.Fatal("ASSINAFY_TEST_EMAIL_PRIMARY is required for embedded signing notifications")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	account, err := c.Accounts.GetWithParams(ctx, "", &models.AccountGetParams{Expand: "signing_features"})
	if err != nil {
		t.Fatalf("Accounts.GetWithParams: %v", err)
	}
	if account.SigningFeatures == nil {
		t.Fatal("account omitted the requested signing features")
	}
	if !account.SigningFeatures.Embed {
		t.Skip("workspace plan does not include embedded signing")
	}
	origins, err := c.Embed.ListOrigins(ctx, "")
	if err != nil {
		t.Fatalf("Embed.ListOrigins: %v", err)
	}
	if len(origins) >= 10 {
		t.Skip("all embed origin slots are occupied")
	}
	name := "go-sdk-embed-" + time.Now().UTC().Format("20060102t150405.000000000")
	originURL := "https://" + strings.ReplaceAll(name, ".", "-") + ".example.com"
	origin, err := c.Embed.CreateOrigin(ctx, "", &models.EmbedOriginRequest{Origin: originURL})
	if err != nil {
		t.Fatalf("Embed.CreateOrigin: %v", err)
	}
	originDeleted := false
	t.Cleanup(func() {
		if originDeleted {
			return
		}
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		if err := c.Embed.DeleteOrigin(cleanCtx, "", origin.ID); err != nil {
			t.Errorf("cleanup temporary embed origin: %v", err)
		}
	})
	if origin.ID == "" || origin.Origin != originURL {
		t.Fatal("created origin omitted its ID or normalized origin")
	}
	updatedURL := strings.Replace(originURL, "https://", "https://updated-", 1)
	updated, err := c.Embed.UpdateOrigin(ctx, "", origin.ID, &models.EmbedOriginRequest{Origin: updatedURL})
	if err != nil || updated.ID != origin.ID || updated.Origin != updatedURL {
		t.Fatalf("Embed.UpdateOrigin did not retain the requested origin: %v", err)
	}

	signer := integrationEmailSigner(ctx, t, c, email)
	doc := uploadReadyDoc(ctx, t, c, "", true)
	zero := 0.0
	body := &models.CreateEnvelopeRequest{
		Name: name, IdempotencyKey: name, MaxTotalCredits: &zero,
		Signers:   []models.EnvelopeSigner{{ID: signer.ID, VerificationMethod: models.VerificationMethodEmail, NotificationMethods: []string{models.NotificationMethodEmail}, Embedded: true}},
		Documents: []models.EnvelopeDocumentRequest{{ID: doc.ID, Method: models.MethodVirtual, SignerIDs: []string{signer.ID}}},
	}
	estimate, err := c.Envelopes.EstimateCost(ctx, "", body)
	var apiErr *sdkerrors.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden && apiErr.Code == "PlanRestricted" {
		t.Skip("origin lifecycle completed; workspace plan does not include envelopes for disposable signing")
	}
	if err != nil {
		t.Fatalf("Envelopes.EstimateCost: %v", err)
	}
	if estimate.TotalCredits != 0 || !estimate.HasSufficientResources {
		t.Skip("origin lifecycle completed; embedded envelope requires resources outside the zero-credit test")
	}
	envelope, err := c.Envelopes.Create(ctx, "", body)
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict && apiErr.Code == "CreditCeilingExceeded" {
		t.Skip("embedded envelope exceeded the zero-credit ceiling")
	}
	if err != nil {
		t.Fatalf("Envelopes.Create: %v", err)
	}
	envelopeDeleted := false
	t.Cleanup(func() {
		if envelopeDeleted {
			return
		}
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanCancel()
		if _, err := c.Envelopes.Cancel(cleanCtx, "", envelope.ID); err != nil {
			t.Errorf("cleanup embedded envelope cancellation: %v", err)
			return
		}
		if err := c.Envelopes.Delete(cleanCtx, "", envelope.ID); err != nil {
			t.Errorf("cleanup embedded envelope deletion: %v", err)
		}
	})
	if len(envelope.SigningURLs) != 1 || !envelope.SigningURLs[0].Embedded || envelope.SigningURLs[0].URL != "" {
		t.Fatal("embedded signer must have an embedded flag without a public signing link")
	}
	read, err := c.Documents.Get(ctx, doc.ID)
	if err != nil || read.Assignment == nil || read.Assignment.ID == "" {
		t.Fatalf("document omitted its envelope assignment: %v", err)
	}
	session, err := c.Embed.CreateSession(ctx, doc.ID, read.Assignment.ID, signer.ID)
	if err != nil {
		t.Fatalf("Embed.CreateSession failed (%T)", err)
	}
	sessionURL, err := url.Parse(session.URL)
	if err != nil || sessionURL.Scheme != "https" || sessionURL.Host == "" || !strings.HasPrefix(sessionURL.Path, "/embed/") {
		t.Fatal("embed session URL has an unsupported shape")
	}
	token := strings.TrimPrefix(sessionURL.Path, "/embed/")
	if decoded, err := hex.DecodeString(token); err != nil || len(decoded) != 32 || session.ExpiresAt.IsZero() {
		t.Fatal("embed session omitted its token or expiry")
	}
	exchanged, err := c.Embed.ExchangeSession(ctx, token, &models.ExchangeEmbedSessionRequest{ParentOrigin: updatedURL})
	if err != nil {
		t.Fatalf("Embed.ExchangeSession failed (%T)", err)
	}
	if exchanged.SignerAccessCode == "" || exchanged.SignerID != signer.ID || exchanged.DocumentID != doc.ID || exchanged.VerificationMethod != models.VerificationMethodEmail {
		t.Fatal("exchange omitted the expected signer context")
	}
	if _, err := c.Embed.ExchangeSession(ctx, token, nil); !sdkerrors.IsStatusCode(err, http.StatusGone) {
		t.Fatalf("replayed embed session did not return 410 (%T)", err)
	}
	if theme, err := c.Signers.GetTheme(ctx, exchanged.SignerAccessCode); err != nil || theme.AccountName == "" {
		t.Fatalf("signer theme failed or omitted workspace name (%T)", err)
	}
	if self, err := c.Signers.GetSelf(ctx, exchanged.SignerAccessCode); err != nil || self.ID != signer.ID {
		t.Fatalf("signer self failed or omitted signer identity (%T)", err)
	}
	if _, err := c.Envelopes.Cancel(ctx, "", envelope.ID); err != nil {
		t.Fatalf("Envelopes.Cancel: %v", err)
	}
	if err := c.Envelopes.Delete(ctx, "", envelope.ID); err != nil {
		t.Fatalf("Envelopes.Delete: %v", err)
	}
	envelopeDeleted = true
	if err := c.Embed.DeleteOrigin(ctx, "", origin.ID); err != nil {
		t.Fatalf("Embed.DeleteOrigin: %v", err)
	}
	originDeleted = true
	remaining, err := c.Embed.ListOrigins(ctx, "")
	if err != nil {
		t.Fatalf("Embed.ListOrigins after deletion: %v", err)
	}
	for _, item := range remaining {
		if item.ID == origin.ID {
			t.Fatal("deleted temporary origin remains listed")
		}
	}
}
