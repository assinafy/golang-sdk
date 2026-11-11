package assinafy

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

// TestIntegrationEmailSigningCompletion waits for the invited participant to
// complete two documents through email, then checks certification and downloads.
// It requires dashboard support for navigating every document in an envelope.
func TestIntegrationEmailSigningCompletion(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_EMAIL_SIGNING_TEST") != "1" {
		t.Skip("set ASSINAFY_RUN_EMAIL_SIGNING_TEST=1 for interactive email signing")
	}
	if !isProductionBaseURL(os.Getenv("ASSINAFY_BASE_URL")) {
		t.Skip("interactive email signing requires the production API")
	}
	email := strings.TrimSpace(os.Getenv("ASSINAFY_TEST_EMAIL_PRIMARY"))
	if email == "" {
		t.Fatal("ASSINAFY_TEST_EMAIL_PRIMARY is required for interactive signing")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 22*time.Minute)
	defer cancel()
	signer := integrationEmailSigner(ctx, t, c, email)
	name := "go-sdk-email-signing-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	first := uploadEmailSigningDocument(ctx, t, c, name+"-first.pdf")
	second := uploadEmailSigningDocument(ctx, t, c, name+"-second.pdf")
	message := "Assinafy Go SDK integration test. Please sign both test documents."
	zero := 0.0
	body := &models.CreateEnvelopeRequest{
		Name: name, Message: &message, IdempotencyKey: name, MaxTotalCredits: &zero,
		Signers: []models.EnvelopeSigner{{ID: signer.ID, VerificationMethod: models.VerificationMethodEmail, NotificationMethods: []string{models.NotificationMethodEmail}}},
		Documents: []models.EnvelopeDocumentRequest{
			{ID: first.ID, Method: models.MethodVirtual, SignerIDs: []string{signer.ID}},
			{ID: second.ID, Method: models.MethodVirtual, SignerIDs: []string{signer.ID}},
		},
	}
	estimate, err := c.Envelopes.EstimateCost(ctx, "", body)
	var apiErr *sdkerrors.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden && apiErr.Code == "PlanRestricted" {
		t.Skip("the workspace plan does not allow envelopes")
	}
	if err != nil {
		t.Fatalf("Envelopes.EstimateCost: %v", err)
	}
	if estimate.TotalCredits != 0 || !estimate.HasSufficientResources {
		t.Skipf("email signing exceeds the zero-credit test: credits=%v, sufficient=%v", estimate.TotalCredits, estimate.HasSufficientResources)
	}
	envelope, err := c.Envelopes.Create(ctx, "", body)
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict && apiErr.Code == "CreditCeilingExceeded" {
		t.Skip("email signing exceeded the zero-credit ceiling")
	}
	if err != nil {
		t.Fatalf("Envelopes.Create: %v", err)
	}
	if envelope == nil || envelope.ID == "" {
		t.Fatal("envelope creation returned no identifier")
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanCancel()
		if _, err := c.Envelopes.Cancel(cleanCtx, "", envelope.ID); err != nil {
			t.Errorf("cleanup email envelope cancellation: %v", err)
			return
		}
		if err := c.Envelopes.Delete(cleanCtx, "", envelope.ID); err != nil {
			t.Errorf("cleanup email envelope deletion: %v", err)
		}
	})
	t.Log("Email invitation sent. Complete both test documents through the received invitation; waiting up to twenty minutes.")

	waitCtx, waitCancel := context.WithTimeout(ctx, 20*time.Minute)
	defer waitCancel()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	completed := make(map[string]*models.Document, 2)
	for len(completed) != 2 {
		for _, id := range []string{first.ID, second.ID} {
			if completed[id] != nil {
				continue
			}
			doc, err := c.Documents.Get(waitCtx, id)
			if err != nil {
				t.Fatalf("read document while waiting for signature: %v", err)
			}
			switch doc.Status {
			case models.StatusCertificated:
				completed[id] = doc
				t.Logf("Certified test documents: %d of 2", len(completed))
			case models.StatusFailed, models.StatusExpired, models.StatusRejectedBySigner, models.StatusRejectedByUser:
				t.Fatalf("test document ended in %s", doc.Status)
			}
		}
		if len(completed) == 2 {
			break
		}
		select {
		case <-waitCtx.Done():
			t.Fatal("email signing did not finish within twenty minutes")
		case <-ticker.C:
		}
	}

	for _, artifact := range []string{"original", "certificated"} {
		archive, err := c.Envelopes.Download(ctx, "", envelope.ID, artifact)
		if err != nil {
			t.Fatalf("Envelopes.Download(%s): %v", artifact, err)
		}
		zipFile, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			t.Fatalf("invalid %s ZIP: %v", artifact, err)
		}
		if len(zipFile.File) != 2 {
			t.Fatalf("%s ZIP contains %d entries, want two documents", artifact, len(zipFile.File))
		}
	}
	for _, id := range []string{first.ID, second.ID} {
		doc := completed[id]
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
		activities, err := c.Documents.ActivitiesWithParams(ctx, id, &models.ListParams{PerPage: 100})
		if err != nil {
			t.Fatalf("Documents.ActivitiesWithParams: %v", err)
		}
		signed, certified := false, false
		for _, activity := range activities.Data {
			signed = signed || activity.Event == "signer_signed_document"
			certified = certified || activity.Event == "document_ready"
		}
		if !signed || !certified {
			t.Fatal("document activity omitted signing or certification")
		}
	}
	t.Log("Both email signatures, certification, public verification and ZIP downloads completed.")
}

// uploadEmailSigningDocument names each PDF before inviting the signer and keeps
// certified records, which the API does not permit clients to delete.
func uploadEmailSigningDocument(ctx context.Context, t *testing.T, c *Client, name string) *models.Document {
	t.Helper()
	doc := uploadReadyDoc(ctx, t, c, "", false)
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		current, err := c.Documents.Get(cleanCtx, doc.ID)
		if err != nil {
			t.Errorf("cleanup read document %s: %v", doc.ID, err)
			return
		}
		if current.Status == models.StatusCertificated {
			t.Logf("Retaining certified document %s: the API does not permit its deletion", doc.ID)
			return
		}
		if err := c.Documents.Delete(cleanCtx, doc.ID); err != nil {
			// Certification can finish between the status read and deletion.
			current, readErr := c.Documents.Get(cleanCtx, doc.ID)
			if readErr == nil && current.Status == models.StatusCertificated {
				t.Logf("Retaining certified document %s: the API does not permit its deletion", doc.ID)
				return
			}
			t.Errorf("cleanup delete unsigned document %s: %v", doc.ID, err)
		}
	})
	renamed, err := c.Documents.Rename(ctx, doc.ID, name)
	if err != nil {
		t.Fatalf("Documents.Rename: %v", err)
	}
	if renamed.ID != doc.ID || renamed.Name == "" {
		t.Fatal("renamed test document omitted its identifier or name")
	}
	return renamed
}
