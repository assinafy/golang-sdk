package assinafy

import (
	"bytes"
	"context"
	stderrors "errors"
	"image/png"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

// TestIntegrationWebhookSubscriptionLifecycle updates the subscription, reads it
// back, inactivates it, and restores the original configuration. It keeps
// is_active=false throughout so no deliveries are triggered.
func TestIntegrationWebhookSubscriptionLifecycle(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	original, err := c.Webhooks.GetSubscription(ctx, "")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if original.URL == nil || original.Email == nil || *original.URL == "" || *original.Email == "" {
		t.Skip("webhook subscription has no losslessly restorable URL/email")
	}
	// Restore the original configuration when the test finishes.
	t.Cleanup(func() {
		restore := &models.UpdateWebhookSubscriptionRequest{
			Events:   original.Events,
			IsActive: original.IsActive,
			URL:      *original.URL,
			Email:    *original.Email,
		}
		if _, err := c.Webhooks.UpdateSubscription(context.Background(), "", restore); err != nil {
			t.Errorf("cleanup restore subscription: %v", err)
		}
	})

	updated, err := c.Webhooks.UpdateSubscription(ctx, "", &models.UpdateWebhookSubscriptionRequest{
		Events:   []string{"document_ready"},
		IsActive: false,
		URL:      "https://example.com/assinafy-go-sdk-webhook",
		Email:    "sdk-webhook@example.test",
	})
	if err != nil {
		t.Fatalf("UpdateSubscription: %v", err)
	}
	if updated.URL == nil || *updated.URL != "https://example.com/assinafy-go-sdk-webhook" {
		t.Errorf("UpdateSubscription url = %v", updated.URL)
	}

	got, err := c.Webhooks.GetSubscription(ctx, "")
	if err != nil {
		t.Fatalf("GetSubscription after update: %v", err)
	}
	if got.Email == nil || *got.Email != "sdk-webhook@example.test" {
		t.Errorf("subscription email = %v, want sdk-webhook@example.test", got.Email)
	}

	inactivated, err := c.Webhooks.Inactivate(ctx, "")
	if err != nil {
		t.Fatalf("Inactivate: %v", err)
	}
	if inactivated.IsActive {
		t.Error("expected subscription to be inactive after Inactivate")
	}
}

func TestIntegrationAccountStats(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := c.Accounts.Stats(ctx, "", nil)
	if err != nil {
		skipIfSandboxRouteIsUnavailable(t, err)
		t.Fatalf("Accounts.Stats: %v", err)
	}
	checkIntegrationStats(t, rows)
}

func TestIntegrationUserGetSelf(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := c.Users.GetSelf(ctx)
	if err != nil {
		t.Fatalf("Users.GetSelf: %v", err)
	}
	if user.ID == "" || user.Email == "" {
		t.Fatalf("Users.GetSelf returned an incomplete user: %+v", user)
	}
}

func TestIntegrationUserStats(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := c.Users.Stats(ctx, nil)
	if err != nil {
		skipIfSandboxRouteIsUnavailable(t, err)
		t.Fatalf("Users.Stats: %v", err)
	}
	checkIntegrationStats(t, rows)
}

func TestIntegrationUserGetNotificationPreferences(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prefs, err := c.Users.GetNotificationPreferences(ctx)
	if err != nil {
		skipIfSandboxRouteIsUnavailable(t, err)
		t.Fatalf("Users.GetNotificationPreferences: %v", err)
	}
	keys := []models.NotificationPreference{
		models.NotificationPreferenceDocumentCompleted,
		models.NotificationPreferenceSignerDeclined,
		models.NotificationPreferenceDocumentCancelled,
		models.NotificationPreferenceDocumentAboutToExpire,
		models.NotificationPreferenceDocumentExpired,
		models.NotificationPreferenceDocumentExpirationReset,
		models.NotificationPreferenceDocumentProcessingFailed,
		models.NotificationPreferenceTemplateProcessingFailed,
		models.NotificationPreferenceSignerWhatsAppFailed,
		models.NotificationPreferenceContractRenewalNotice,
		models.NotificationPreferenceContractExpiring,
		models.NotificationPreferenceContractExpired,
		models.NotificationPreferenceContractRenewed,
	}
	for _, key := range keys {
		if _, ok := prefs[key]; !ok {
			t.Errorf("notification preferences missing %q: %v", key, prefs)
		}
	}
	if len(prefs) != len(keys) {
		t.Errorf("notification preferences returned %d keys, want %d: %v", len(prefs), len(keys), prefs)
	}
}

func TestIntegrationUserNotificationPreferenceRoundTrip(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const key = models.NotificationPreferenceSignerWhatsAppFailed
	original, err := c.Users.GetNotificationPreferences(ctx)
	if err != nil {
		skipIfSandboxRouteIsUnavailable(t, err)
		t.Fatalf("Users.GetNotificationPreferences: %v", err)
	}
	originalValue, ok := original[key]
	if !ok {
		t.Fatalf("notification preferences missing %q: %v", key, original)
	}

	restored := false
	t.Cleanup(func() {
		if restored {
			return
		}
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		if _, err := c.Users.UpdateNotificationPreferences(cleanCtx, models.NotificationPreferences{key: originalValue}); err != nil {
			t.Errorf("cleanup: could not restore notification preference %q: %v", key, err)
		}
	})

	want := !originalValue
	updated, err := c.Users.UpdateNotificationPreferences(ctx, models.NotificationPreferences{key: want})
	if err != nil {
		t.Fatalf("Users.UpdateNotificationPreferences: %v", err)
	}
	if got, ok := updated[key]; !ok || got != want {
		t.Errorf("updated preference %q = %t, want %t", key, got, want)
	}

	got, err := c.Users.GetNotificationPreferences(ctx)
	if err != nil {
		t.Fatalf("Users.GetNotificationPreferences after update: %v", err)
	}
	if value, ok := got[key]; !ok || value != want {
		t.Errorf("persisted preference %q = %t, want %t", key, value, want)
	}

	final, err := c.Users.UpdateNotificationPreferences(ctx, models.NotificationPreferences{key: originalValue})
	if err != nil {
		t.Fatalf("restore notification preference %q: %v", key, err)
	}
	if value, ok := final[key]; !ok || value != originalValue {
		t.Errorf("restored preference %q = %t, want %t", key, value, originalValue)
		return
	}
	restored = true
}

func TestIntegrationAccountAdminLifecycle(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_ACCOUNT_ADMIN_TESTS") != "1" {
		t.Skip("set ASSINAFY_RUN_ACCOUNT_ADMIN_TESTS=1 to run the account administration flow")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	name := "go-sdk-admin-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	account, err := c.Accounts.Create(ctx, &models.CreateAccountRequest{Name: name})
	if err != nil {
		t.Fatalf("Accounts.Create: %v", err)
	}

	deleted := false
	t.Cleanup(func() {
		if deleted || account.ID == "" {
			return
		}
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		if err := c.Accounts.Delete(cleanCtx, account.ID, true); err != nil {
			t.Errorf("cleanup: could not delete account %s: %v", account.ID, err)
		}
	})
	if account.ID == "" || account.Name != name {
		t.Fatalf("Accounts.Create returned %+v", account)
	}

	if err := c.Accounts.UploadLogo(ctx, account.ID, onePixelPNG(t), "logo.png"); err != nil {
		t.Fatalf("Accounts.UploadLogo: %v", err)
	}
	downloaded, err := c.Accounts.DownloadLogo(ctx, account.ID)
	if err != nil {
		t.Fatalf("Accounts.DownloadLogo: %v", err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(downloaded))
	if err != nil {
		t.Fatalf("decode downloaded logo: %v", err)
	}
	if config.Width != 1 || config.Height != 1 {
		t.Errorf("downloaded logo dimensions = %dx%d, want 1x1", config.Width, config.Height)
	}

	if err := c.Accounts.DeleteLogo(ctx, account.ID); err != nil {
		t.Fatalf("Accounts.DeleteLogo: %v", err)
	}
	inactive := false
	endpoint, err := c.Webhooks.CreateEndpoint(ctx, account.ID, &models.CreateWebhookEndpointRequest{
		URL: "https://example.com/assinafy-go-sdk-webhook", Email: "ops@example.com",
		Events: []string{"document_ready"}, IsActive: &inactive, SigningEnabled: true,
	})
	if err != nil {
		t.Fatalf("Webhooks.CreateEndpoint: %v", err)
	}
	secret, err := c.Webhooks.GetEndpointSecret(ctx, account.ID, endpoint.ID)
	if err != nil || !strings.HasPrefix(secret, "whsec_") {
		t.Fatalf("Webhooks.GetEndpointSecret: has prefix=%v, error=%v", strings.HasPrefix(secret, "whsec_"), err)
	}
	if endpoint.Secret != "" && endpoint.Secret != secret {
		t.Error("creation secret does not equal the endpoint signing secret")
	}
	updatedEndpoint, err := c.Webhooks.UpdateEndpoint(ctx, account.ID, endpoint.ID, &models.UpdateWebhookEndpointRequest{Events: []string{}})
	if err != nil {
		t.Fatalf("Webhooks.UpdateEndpoint: %v", err)
	}
	if updatedEndpoint.Events == nil || len(updatedEndpoint.Events) != 0 {
		t.Error("empty Events did not clear the endpoint event selection")
	}
	if err := c.Webhooks.DeleteEndpoint(ctx, account.ID, endpoint.ID); err != nil {
		t.Fatalf("Webhooks.DeleteEndpoint: %v", err)
	}
	if err := c.Accounts.Delete(ctx, account.ID, false); err != nil {
		t.Fatalf("Accounts.Delete: %v", err)
	}
	deleted = true
}

// TestIntegrationAccountsList lists the accounts the credential belongs to and
// checks the slim item shape decodes.
func TestIntegrationAccountsList(t *testing.T) {
	c, accountID := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accts, err := c.Accounts.List(ctx)
	if err != nil {
		t.Fatalf("Accounts.List: %v", err)
	}
	found := false
	for _, a := range accts {
		if a.ID == accountID {
			found = true
		}
		if a.ID == "" || a.Name == "" {
			t.Errorf("account item missing id/name: %+v", a)
		}
	}
	if !found {
		t.Errorf("configured account %q not present in Accounts.List result", accountID)
	}
}

// TestIntegrationAccountGet fetches the configured account and checks the
// single-account shape.
func TestIntegrationAccountGet(t *testing.T) {
	c, accountID := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acct, err := c.Accounts.Get(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.Get: %v", err)
	}
	if acct.ID != accountID {
		t.Errorf("account id = %q, want %q", acct.ID, accountID)
	}
	if acct.CreatedAt.IsZero() {
		t.Error("expected created_at to be populated")
	}
}

// TestIntegrationAccountTheme fetches the account branding theme.
func TestIntegrationAccountTheme(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	theme, err := c.Accounts.GetTheme(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.GetTheme: %v", err)
	}
	if theme.AccountName == "" {
		t.Errorf("theme account_name empty: %+v", theme)
	}
}

// TestIntegrationAccountLogoDownload confirms the logo endpoint is reachable. A
// workspace without a logo returns a 404 APIError, which is an acceptable
// outcome; a configured logo returns image bytes.
func TestIntegrationAccountLogoDownload(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	img, err := c.Accounts.DownloadLogo(ctx, "")
	if err != nil {
		if !sdkerrors.IsStatusCode(err, 404) {
			t.Fatalf("Accounts.DownloadLogo: unexpected error: %v", err)
		}
		return // no logo configured — endpoint reachable, 404 is fine
	}
	if len(img) == 0 {
		t.Error("logo download returned no bytes")
	}
}

// TestIntegrationAccountUpdateRoundTrip performs a non-destructive update by
// writing the account's existing name back to it and confirming the response.
func TestIntegrationAccountUpdateRoundTrip(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	current, err := c.Accounts.Get(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.Get: %v", err)
	}
	name := current.Name
	updated, err := c.Accounts.Update(ctx, "", &models.UpdateAccountRequest{Name: &name})
	if err != nil {
		t.Fatalf("Accounts.Update: %v", err)
	}
	if updated.Name != name {
		t.Errorf("Accounts.Update name = %q, want %q", updated.Name, name)
	}
}

func TestIntegrationWebhookSubscription(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := c.Webhooks.GetSubscription(ctx, ""); err != nil {
		t.Fatalf("Webhooks.GetSubscription: %v", err)
	}
}

func TestIntegrationGetAPIKey(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	got, err := c.Authentication.GetAPIKey(ctx)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got.APIKey == nil || !strings.Contains(*got.APIKey, "*") {
		t.Fatalf("expected masked api key, got %+v", got)
	}
}

// TestIntegrationWebhookDispatches lists webhook delivery attempts. An empty
// result is valid; the test only proves the request encoding and response
// decoding succeed against the live API.
func TestIntegrationWebhookDispatches(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := c.Webhooks.ListDispatches(ctx, "", &models.WebhookDispatchListParams{PerPage: 5}); err != nil {
		t.Fatalf("Webhooks.ListDispatches: %v", err)
	}
}

// TestIntegrationWebhookEndpoints reads the account's endpoints, then exercises
// the signing secret on a temporary endpoint. When the plan has no free slot
// (403), it borrows an existing inactive endpoint that does not sign
// deliveries, enables signing on it and restores it afterwards, so no live
// receiver ever sees a changed secret.
func TestIntegrationWebhookEndpoints(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	endpoints, err := c.Webhooks.ListEndpoints(ctx, "")
	if err != nil {
		t.Fatalf("Webhooks.ListEndpoints: %v", err)
	}
	for _, endpoint := range endpoints {
		got, err := c.Webhooks.GetEndpoint(ctx, "", endpoint.ID)
		if err != nil {
			t.Fatalf("Webhooks.GetEndpoint(%s): %v", endpoint.ID, err)
		}
		if got.ID != endpoint.ID || got.URL == "" {
			t.Errorf("Webhooks.GetEndpoint = %+v", got)
		}
	}

	inactive := false
	target, err := c.Webhooks.CreateEndpoint(ctx, "", &models.CreateWebhookEndpointRequest{
		URL:            "https://example.com/assinafy-go-sdk-" + time.Now().UTC().Format("20060102T150405Z"),
		Email:          "signer@example.com",
		Events:         []string{"document_ready"},
		Name:           "go-sdk-test",
		IsActive:       &inactive,
		SigningEnabled: true,
	})
	switch {
	case err == nil:
		t.Cleanup(func() {
			if err := c.Webhooks.DeleteEndpoint(context.Background(), "", target.ID); err != nil {
				t.Errorf("cleanup Webhooks.DeleteEndpoint(%s): %v", target.ID, err)
			}
		})
	case sdkerrors.IsStatusCode(err, 403):
		t.Logf("no free webhook endpoint slot (%d in use): %v", len(endpoints), err)
		target = borrowUnsignedEndpoint(ctx, t, c, endpoints)
	default:
		t.Fatalf("Webhooks.CreateEndpoint: %v", err)
	}
	if !target.SigningEnabled || target.IsActive {
		t.Fatalf("endpoint under test = %+v, want inactive with signing enabled", target)
	}

	secret, err := c.Webhooks.GetEndpointSecret(ctx, "", target.ID)
	if err != nil || !strings.HasPrefix(secret, "whsec_") {
		t.Fatalf("Webhooks.GetEndpointSecret: whsec_ prefix=%v, %v", strings.HasPrefix(secret, "whsec_"), err)
	}
	rotated, err := c.Webhooks.RotateEndpointSecret(ctx, "", target.ID)
	if err != nil || !strings.HasPrefix(rotated, "whsec_") || rotated == secret {
		t.Fatalf("Webhooks.RotateEndpointSecret: changed=%v, %v", rotated != secret, err)
	}
}

// borrowUnsignedEndpoint enables signing on an inactive endpoint that does not
// sign deliveries and restores it when the test ends. It skips the test when
// no such endpoint exists.
func borrowUnsignedEndpoint(ctx context.Context, t *testing.T, c *Client, endpoints []models.WebhookEndpoint) *models.WebhookEndpoint {
	t.Helper()
	for _, endpoint := range endpoints {
		if endpoint.IsActive || endpoint.SigningEnabled {
			continue
		}
		enabled, disabled := true, false
		updated, err := c.Webhooks.UpdateEndpoint(ctx, "", endpoint.ID, &models.UpdateWebhookEndpointRequest{SigningEnabled: &enabled})
		if err != nil {
			t.Fatalf("Webhooks.UpdateEndpoint(signing_enabled=true): %v", err)
		}
		t.Cleanup(func() {
			if _, err := c.Webhooks.UpdateEndpoint(context.Background(), "", endpoint.ID, &models.UpdateWebhookEndpointRequest{SigningEnabled: &disabled}); err != nil {
				t.Errorf("cleanup Webhooks.UpdateEndpoint(signing_enabled=false): %v", err)
			}
		})
		return updated
	}
	t.Skip("no free endpoint slot and no inactive unsigned endpoint to borrow")
	return nil
}

// TestIntegrationRateLimitHeaders checks that a failed call carries the
// X-Rate-Limit-* quota on its APIError.
func TestIntegrationRateLimitHeaders(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := c.Documents.Get(ctx, "000000000000000000000000000")
	var apiErr *sdkerrors.APIError
	if !stderrors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("Documents.Get(missing) = %v, want a 404 APIError", err)
	}
	if limit := apiErr.RateLimit(); limit.Limit <= 0 || limit.Reset <= 0 {
		t.Errorf("RateLimit() = %+v, want the X-Rate-Limit-* headers", limit)
	}
}
