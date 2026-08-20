package assinafy

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
)

func skipIfSandboxRouteIsUnavailable(t *testing.T, err error) {
	t.Helper()
	if os.Getenv("ASSINAFY_ALLOW_MISSING_DOCUMENTED_ROUTES") != "1" {
		return
	}
	baseURL := os.Getenv("ASSINAFY_BASE_URL")
	if baseURL == "" {
		baseURL = SandboxBaseURL
	}
	if strings.TrimRight(baseURL, "/") == SandboxBaseURL && sdkerrors.IsStatusCode(err, 404) {
		t.Skipf("current sandbox deployment does not expose this documented route: %v", err)
	}
}

func checkIntegrationStats(t *testing.T, rows []models.DocumentStatsRow) {
	t.Helper()
	if len(rows) == 0 {
		t.Fatal("expected at least one zero-filled statistics row")
	}
	for _, row := range rows {
		if row.Period == "" {
			t.Errorf("statistics row has an empty period: %+v", row)
		}
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

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 32, G: 114, B: 185, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
	return out.Bytes()
}

func TestIntegrationAccountAdminLifecycle(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_ACCOUNT_ADMIN_TESTS") != "1" {
		t.Skip("set ASSINAFY_RUN_ACCOUNT_ADMIN_TESTS=1 to run the account administration flow")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	name := "go-sdk-admin-audit-" + time.Now().UTC().Format("20060102T150405.000000000Z")
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
