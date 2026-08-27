package assinafy

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
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
		Email:    "sdk-webhook@example.com",
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
	if got.Email == nil || *got.Email != "sdk-webhook@example.com" {
		t.Errorf("subscription email = %v, want sdk-webhook@example.com", got.Email)
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
