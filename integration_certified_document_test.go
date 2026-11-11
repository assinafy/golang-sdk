package assinafy

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

// TestIntegrationCertifiedDocumentArtifacts reads an existing certified document
// without changing it or sending notifications. The document ID is an explicit
// opt-in through ASSINAFY_TEST_CERTIFIED_DOCUMENT_ID.
func TestIntegrationCertifiedDocumentArtifacts(t *testing.T) {
	id := strings.TrimSpace(os.Getenv("ASSINAFY_TEST_CERTIFIED_DOCUMENT_ID"))
	if id == "" {
		t.Skip("set ASSINAFY_TEST_CERTIFIED_DOCUMENT_ID to read certified artifacts")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	doc, err := c.Documents.Get(ctx, id)
	if err != nil {
		t.Fatalf("Documents.Get: %v", err)
	}
	if doc.Status != models.StatusCertificated {
		t.Fatalf("selected document has status %s, want certificated", doc.Status)
	}
	for _, artifact := range []string{"original", "certificated", "certificate-page", "bundle"} {
		t.Run(artifact, func(t *testing.T) {
			data, err := c.Documents.Download(ctx, id, artifact)
			if err != nil {
				t.Fatalf("Documents.Download(%s): %v", artifact, err)
			}
			if artifact != "bundle" {
				if !bytes.HasPrefix(data, []byte("%PDF")) {
					t.Fatal("downloaded artifact is not a PDF")
				}
				return
			}
			archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatalf("invalid artifact bundle: %v", err)
			}
			if len(archive.File) < 3 {
				t.Fatalf("artifact bundle has %d entries, want at least three", len(archive.File))
			}
		})
	}
	if doc.VerificationCode == nil || *doc.VerificationCode == "" {
		t.Fatal("certified document has no verification code")
	}
	verified, err := c.Documents.Verify(ctx, *doc.VerificationCode)
	if err != nil {
		t.Fatalf("Documents.Verify: %v", err)
	}
	if !verified.IsValid || verified.ID == nil || *verified.ID != id {
		t.Fatal("certified document did not pass public verification")
	}
	if verified.CertificatedAt == nil || verified.CertificatedAt.IsZero() || verified.VerifiedAt.IsZero() {
		t.Error("public verification omitted certification or verification timestamp")
	}
	signed, certified := false, false
	for page := 1; ; page++ {
		activities, err := c.Documents.ActivitiesWithParams(ctx, id, &models.ListParams{Page: page, PerPage: 100})
		if err != nil {
			t.Fatalf("Documents.ActivitiesWithParams: %v", err)
		}
		for _, activity := range activities.Data {
			signed = signed || activity.Event == "signer_signed_document"
			certified = certified || activity.Event == "document_ready"
		}
		if page >= activities.Pagination.PageCount {
			break
		}
	}
	if !signed || !certified {
		t.Error("document activity omitted signing or certification")
	}
	t.Run("signed-thumbnail", func(t *testing.T) {
		checkCertifiedRender(ctx, t, func() ([]byte, error) {
			return c.Documents.ThumbnailWithArtifact(ctx, id, "certificated")
		})
	})
	t.Run("signed-page", func(t *testing.T) {
		if len(doc.Pages) == 0 {
			t.Fatal("certified document has no page metadata")
		}
		checkCertifiedRender(ctx, t, func() ([]byte, error) {
			return c.Documents.DownloadPageWithArtifact(ctx, id, doc.Pages[0].ID, "certificated")
		})
	})
}

// checkCertifiedRender permits the documented temporary rendering-capacity 503
// without treating unavailable renders as a successful endpoint check.
func checkCertifiedRender(ctx context.Context, t *testing.T, download func() ([]byte, error)) {
	t.Helper()
	for attempt := 0; attempt < 3; attempt++ {
		data, err := download()
		if err == nil {
			if !strings.HasPrefix(http.DetectContentType(data), "image/") {
				t.Fatal("signed render is not an image")
			}
			return
		}
		if !sdkerrors.IsStatusCode(err, http.StatusServiceUnavailable) {
			t.Fatalf("signed render: %v", err)
		}
		if attempt == 2 {
			t.Skip("signed rendering remains unavailable after three attempts (HTTP 503)")
		}
		select {
		case <-ctx.Done():
			t.Fatal("signed render retry exceeded the test deadline")
		case <-time.After(5 * time.Second):
		}
	}
}
