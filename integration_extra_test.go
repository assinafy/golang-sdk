package assinafy

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
)

// uploadReadyDoc uploads the minimal PDF and polls until the document reaches
// metadata_ready, registering a best-effort delete cleanup. It fails the test
// if the document never becomes ready.
func uploadReadyDoc(t *testing.T, c *Client, ctx context.Context) *models.Document {
	t.Helper()
	doc, err := c.Documents.Upload(ctx, "", minimalPDF(), "go-sdk-audit.pdf", nil)
	if err != nil {
		t.Fatalf("Documents.Upload: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for i := 0; i < 8; i++ {
			if err := c.Documents.Delete(cleanCtx, doc.ID); err == nil {
				return
			}
			time.Sleep(2 * time.Second)
		}
		t.Logf("cleanup: could not delete %s within timeout", doc.ID)
	})

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

// TestIntegrationDocumentDownloads exercises the binary artifact and page
// download endpoints against a freshly processed document.
func TestIntegrationDocumentDownloads(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	doc := uploadReadyDoc(t, c, ctx)

	original, err := c.Documents.Download(ctx, doc.ID, "original")
	if err != nil {
		t.Fatalf("Documents.Download(original): %v", err)
	}
	if !bytes.HasPrefix(original, []byte("%PDF")) {
		t.Errorf("original artifact is not a PDF (len=%d)", len(original))
	}

	if len(doc.Pages) == 0 {
		t.Fatal("expected at least one page after metadata_ready")
	}
	page, err := c.Documents.DownloadPage(ctx, doc.ID, doc.Pages[0].ID)
	if err != nil {
		t.Fatalf("Documents.DownloadPage: %v", err)
	}
	if len(page) == 0 {
		t.Error("empty page download")
	}
}

// TestIntegrationDocumentVerifyUnknownHash confirms the verify endpoint decodes
// into VerifyDocumentResult and reports an unknown hash as invalid.
func TestIntegrationDocumentVerifyUnknownHash(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := c.Documents.Verify(ctx, "000000000000000000000000")
	if err != nil {
		t.Fatalf("Documents.Verify: %v", err)
	}
	if res.IsValid {
		t.Error("expected unknown hash to be reported invalid")
	}
	if res.VerifiedAt.IsZero() {
		t.Error("expected verified_at to be populated")
	}
}

// TestIntegrationTemplateGetUnknown confirms the single-template route exists and
// returns a 404 APIError for an unknown template.
func TestIntegrationTemplateGetUnknown(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := c.Templates.Get(ctx, "", "000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
	if !sdkerrors.IsStatusCode(err, 404) {
		t.Errorf("expected 404 APIError, got %v", err)
	}
}

// TestIntegrationDocumentsListWithTags exercises the documents list tags filter.
// An empty result is acceptable; the test only proves the request is accepted.
func TestIntegrationDocumentsListWithTags(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tags, err := c.Tags.List(ctx, "", "")
	if err != nil {
		t.Fatalf("Tags.List: %v", err)
	}
	if len(tags) == 0 {
		t.Skip("workspace has no tags to filter documents by")
	}
	page, err := c.Documents.List(ctx, "", &models.ListParams{PerPage: 5, Tags: []string{tags[0].ID}})
	if err != nil {
		t.Fatalf("Documents.List(tags): %v", err)
	}
	_ = page
}

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
	// Restore the original configuration when the test finishes.
	t.Cleanup(func() {
		restore := &models.UpdateWebhookSubscriptionRequest{
			Events:   original.Events,
			IsActive: false,
		}
		if original.URL != nil {
			restore.URL = *original.URL
		}
		if original.Email != nil {
			restore.Email = *original.Email
		}
		if _, err := c.Webhooks.UpdateSubscription(context.Background(), "", restore); err != nil {
			t.Logf("cleanup restore subscription: %v", err)
		}
	})

	updated, err := c.Webhooks.UpdateSubscription(ctx, "", &models.UpdateWebhookSubscriptionRequest{
		Events:   []string{"document_ready"},
		IsActive: false,
		URL:      "https://example.com/assinafy-go-sdk-audit",
		Email:    "sdk-audit@example.com",
	})
	if err != nil {
		t.Fatalf("UpdateSubscription: %v", err)
	}
	if updated.URL == nil || *updated.URL != "https://example.com/assinafy-go-sdk-audit" {
		t.Errorf("UpdateSubscription url = %v", updated.URL)
	}

	got, err := c.Webhooks.GetSubscription(ctx, "")
	if err != nil {
		t.Fatalf("GetSubscription after update: %v", err)
	}
	if got.Email == nil || *got.Email != "sdk-audit@example.com" {
		t.Errorf("subscription email = %v, want sdk-audit@example.com", got.Email)
	}

	inactivated, err := c.Webhooks.Inactivate(ctx, "")
	if err != nil {
		t.Fatalf("Inactivate: %v", err)
	}
	if inactivated.IsActive {
		t.Error("expected subscription to be inactive after Inactivate")
	}
}

// findOrCreateSigner returns the workspace signer with the given email, creating
// it (and registering a delete cleanup) only when it does not already exist.
// Signer emails are unique per workspace, so reuse avoids a 400 conflict.
func findOrCreateSigner(t *testing.T, c *Client, ctx context.Context, email string) *models.Signer {
	t.Helper()
	page, err := c.Signers.List(ctx, "", &models.ListParams{Search: email, PerPage: 25})
	if err != nil {
		t.Fatalf("Signers.List(%s): %v", email, err)
	}
	for i := range page.Data {
		if page.Data[i].Email != nil && *page.Data[i].Email == email {
			return &page.Data[i]
		}
	}

	e := email
	signer, err := c.Signers.Create(ctx, "", &models.CreateSignerRequest{
		FullName: "Go SDK Audit Signer",
		Email:    &e,
	})
	if err != nil {
		t.Fatalf("Signers.Create(%s): %v", email, err)
	}
	id := signer.ID
	t.Cleanup(func() {
		if err := c.Signers.Delete(context.Background(), "", id); err != nil {
			t.Logf("cleanup Signers.Delete(%s): %v", id, err)
		}
	})
	return signer
}

// TestIntegrationAssignmentLifecycle exercises the full virtual-assignment flow
// against the live API: create signers, create the assignment, estimate a
// resend, reset the expiration, and list WhatsApp notifications. It SENDS real
// signature-request emails, so it is opt-in via ASSINAFY_RUN_ASSIGNMENT_TESTS=1.
func TestIntegrationAssignmentLifecycle(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_ASSIGNMENT_TESTS") == "" {
		t.Skip("set ASSINAFY_RUN_ASSIGNMENT_TESTS=1 to run the assignment flow (sends emails)")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	doc := uploadReadyDoc(t, c, ctx)

	emails := []string{"bill@febacapital.com", "billm@billm.org"}
	signerRefs := make([]models.SignerReference, 0, len(emails))
	for _, email := range emails {
		signer := findOrCreateSigner(t, c, ctx, email)
		signerRefs = append(signerRefs, models.SignerReference{
			ID:                  signer.ID,
			VerificationMethod:  "Email",
			NotificationMethods: []string{"Email"},
		})
	}

	message := "Assinafy Go SDK audit — please disregard this test request."
	assignment, err := c.Assignments.Create(ctx, doc.ID, &models.CreateAssignmentRequest{
		Method:  models.MethodVirtual,
		Signers: signerRefs,
		Message: &message,
	})
	if err != nil {
		t.Fatalf("Assignments.Create: %v", err)
	}
	if assignment.ID == "" {
		t.Fatal("expected non-empty assignment id")
	}

	estimate, err := c.Assignments.EstimateResendCost(ctx, doc.ID, assignment.ID, signerRefs[0].ID)
	if err != nil {
		t.Fatalf("Assignments.EstimateResendCost: %v", err)
	}
	if estimate.Total != 0 {
		t.Errorf("expected 0-credit email resend estimate, got %v", estimate.Total)
	}

	future := time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02T15:04:05Z")
	if _, err := c.Assignments.ResetExpiration(ctx, doc.ID, assignment.ID, &future); err != nil {
		t.Fatalf("Assignments.ResetExpiration: %v", err)
	}

	if _, err := c.Assignments.ListWhatsAppNotifications(ctx, doc.ID, assignment.ID); err != nil {
		t.Fatalf("Assignments.ListWhatsAppNotifications: %v", err)
	}
}
