package assinafy

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
)

// uploadReadyDoc uploads the minimal PDF and polls until the document reaches
// metadata_ready. It optionally registers a delete cleanup; a disposable-account
// caller can instead rely on account cleanup for active signing flows.
func uploadReadyDoc(ctx context.Context, t *testing.T, c *Client, accountID string, cleanupDocument bool) *models.Document {
	t.Helper()
	doc, err := c.Documents.Upload(ctx, accountID, minimalPDF(), "go-sdk-test.pdf", nil)
	if err != nil {
		t.Fatalf("Documents.Upload: %v", err)
	}
	if cleanupDocument {
		t.Cleanup(func() {
			cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			for i := 0; i < 8; i++ {
				if err := c.Documents.Delete(cleanCtx, doc.ID); err == nil {
					return
				}
				time.Sleep(2 * time.Second)
			}
			t.Errorf("cleanup: could not delete %s within timeout", doc.ID)
		})
	}

	for i := 0; i < 15; i++ {
		got, err := c.Documents.Get(ctx, doc.ID)
		if err != nil {
			t.Fatalf("Documents.Get while polling: %v", err)
		}
		if got.Status == models.StatusMetadataReady {
			return got
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("document %s did not reach metadata_ready", doc.ID)
	return nil
}

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

// integrationClient builds a client from environment variables. The test is
// skipped unless explicitly enabled and credentials are present, so ordinary
// unit-test runs cannot mutate an account accidentally.
//
// ASSINAFY_BASE_URL optionally overrides the API base URL and defaults to the
// sandbox. Production requires ASSINAFY_RUN_PRODUCTION_TESTS=1 because these
// tests create, update, send, and delete real resources.
func integrationClient(t *testing.T) (*Client, string) {
	t.Helper()
	if os.Getenv("ASSINAFY_RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set ASSINAFY_RUN_INTEGRATION_TESTS=1 to run mutating integration tests")
	}
	apiKey := os.Getenv("ASSINAFY_API_KEY")
	accountID := os.Getenv("ASSINAFY_ACCOUNT_ID")
	if apiKey == "" || accountID == "" {
		t.Skip("ASSINAFY_API_KEY and ASSINAFY_ACCOUNT_ID must be set for integration tests")
	}
	baseURL := os.Getenv("ASSINAFY_BASE_URL")
	if baseURL == "" {
		baseURL = SandboxBaseURL
	}
	if isProductionBaseURL(baseURL) && os.Getenv("ASSINAFY_RUN_PRODUCTION_TESTS") != "1" {
		t.Fatal("refusing to run integration tests against production without ASSINAFY_RUN_PRODUCTION_TESTS=1")
	}
	c, err := NewClient(ClientOptions{
		APIKey:    apiKey,
		AccountID: accountID,
		BaseURL:   baseURL,
		// 60s tolerates cold-start latency on the first request to the sandbox
		// (Cloudflare/origin warm-up) that occasionally exceeds a tighter timeout.
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, accountID
}

func isProductionBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && strings.TrimSuffix(strings.ToLower(u.Hostname()), ".") == "api.assinafy.com.br"
}

func TestProductionBaseURLGuard(t *testing.T) {
	for _, raw := range []string{
		DefaultBaseURL,
		"https://API.ASSINAFY.COM.BR:443/v1/",
		"https://api.assinafy.com.br./v1",
	} {
		if !isProductionBaseURL(raw) {
			t.Errorf("isProductionBaseURL(%q) = false", raw)
		}
	}
	for _, raw := range []string{SandboxBaseURL, "https://api.assinafy.com.br.example/v1", "://bad"} {
		if isProductionBaseURL(raw) {
			t.Errorf("isProductionBaseURL(%q) = true", raw)
		}
	}
}

func TestIntegrationDocumentStatuses(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	statuses, err := c.Documents.ListStatuses(ctx)
	if err != nil {
		t.Fatalf("ListStatuses: %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("expected at least one status entry")
	}
	for _, s := range statuses {
		if s.Code == "" {
			t.Errorf("empty code in status entry: %+v", s)
		}
	}
}

func TestIntegrationFieldTypes(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	types, err := c.Fields.ListTypes(ctx)
	if err != nil {
		t.Fatalf("ListTypes: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("expected at least one field type")
	}
}

func TestIntegrationWebhookEventTypes(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, err := c.Webhooks.ListEventTypes(ctx)
	if err != nil {
		t.Fatalf("ListEventTypes: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one event type")
	}
}

// minimalPDF is the smallest spec-conformant PDF we can upload for testing.
// Borrowed from common "smallest valid PDF" references.
func minimalPDF() []byte {
	return []byte(
		"%PDF-1.1\n" +
			"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n" +
			"2 0 obj<</Type/Pages/Count 1/Kids[3 0 R]>>endobj\n" +
			"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Resources<<>>>>endobj\n" +
			"xref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000053 00000 n \n0000000100 00000 n \ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n168\n%%EOF\n",
	)
}
