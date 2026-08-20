package assinafy

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/assinafy/golang-sdk/models"
)

// uploadReadyDoc uploads the minimal PDF and polls until the document reaches
// metadata_ready. It optionally registers a delete cleanup; a disposable-account
// caller can instead rely on account cleanup for active signing flows.
func uploadReadyDoc(t *testing.T, c *Client, ctx context.Context, accountID string, cleanupDocument bool) *models.Document {
	t.Helper()
	doc, err := c.Documents.Upload(ctx, accountID, minimalPDF(), "go-sdk-audit.pdf", nil)
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

// TestIntegrationDocumentDownloads exercises the binary artifact and page
// download endpoints against a freshly processed document.
func TestIntegrationDocumentDownloads(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	doc := uploadReadyDoc(t, c, ctx, "", true)

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
	thumbnail, err := c.Documents.Thumbnail(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Documents.Thumbnail: %v", err)
	}
	if len(thumbnail) == 0 {
		t.Error("empty thumbnail download")
	}
	public, err := c.PublicDocuments.GetDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("PublicDocuments.GetDocument: %v", err)
	}
	if public.ID != doc.ID {
		t.Errorf("PublicDocuments.GetDocument ID = %q, want %q", public.ID, doc.ID)
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

// TestIntegrationTemplateGet exercises the compatibility route using a real
// template returned by the sandbox. This route is not in the public OpenAPI
// document, so a fabricated-ID 404 is not sufficient evidence that it works.
func TestIntegrationTemplateGet(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Templates.List(ctx, "", &models.ListParams{PerPage: 1})
	if err != nil {
		t.Fatalf("Templates.List: %v", err)
	}
	if len(page.Data) == 0 {
		t.Skip("workspace has no template with which to verify Templates.Get")
	}
	template, err := c.Templates.Get(ctx, "", page.Data[0].ID)
	if err != nil {
		t.Fatalf("Templates.Get: %v", err)
	}
	if template.ID != page.Data[0].ID {
		t.Errorf("Templates.Get ID = %q, want %q", template.ID, page.Data[0].ID)
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

// TestIntegrationAssignmentLifecycle exercises the full virtual-assignment flow
// in a disposable account: create signers, create the assignment, estimate a
// resend, reset the expiration, and list WhatsApp notifications. It SENDS real
// signature-request emails, so it is opt-in via ASSINAFY_RUN_ASSIGNMENT_TESTS=1.
func TestIntegrationAssignmentLifecycle(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_ASSIGNMENT_TESTS") != "1" {
		t.Skip("set ASSINAFY_RUN_ASSIGNMENT_TESTS=1 to run the assignment flow (sends emails)")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	account, err := c.Accounts.Create(ctx, &models.CreateAccountRequest{
		Name: "go-sdk-assignment-audit-" + time.Now().UTC().Format("20060102T150405.000000000Z"),
	})
	if err != nil {
		t.Fatalf("Accounts.Create for assignment flow: %v", err)
	}
	deleted := false
	t.Cleanup(func() {
		if deleted {
			return
		}
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanCancel()
		if err := c.Accounts.Delete(cleanCtx, account.ID, true); err != nil {
			t.Errorf("cleanup Accounts.Delete(%s): %v", account.ID, err)
		}
	})

	doc := uploadReadyDoc(t, c, ctx, account.ID, false)

	emails := []string{"bill@febacapital.com", "billm@billm.org"}
	signerRefs := make([]models.SignerReference, 0, len(emails))
	for _, email := range emails {
		email := email
		signer, err := c.Signers.Create(ctx, account.ID, &models.CreateSignerRequest{
			FullName: "Go SDK Audit Signer",
			Email:    &email,
		})
		if err != nil {
			t.Fatalf("Signers.Create(%s): %v", email, err)
		}
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
	if err := c.PublicDocuments.SendTokenByEmail(ctx, doc.ID, emails[0]); err != nil {
		t.Fatalf("PublicDocuments.SendTokenByEmail: %v", err)
	}

	estimate, err := c.Assignments.EstimateResendCost(ctx, doc.ID, assignment.ID, signerRefs[0].ID)
	if err != nil {
		t.Fatalf("Assignments.EstimateResendCost: %v", err)
	}
	if estimate.Total != 0 {
		t.Errorf("expected 0-credit email resend estimate, got %v", estimate.Total)
	}
	if !estimate.HasSufficientCredits && !estimate.HasSufficientResources {
		t.Errorf("resend estimate omitted its sufficiency result: %+v", estimate)
	}
	resent, err := c.Assignments.ResendNotification(ctx, doc.ID, assignment.ID, signerRefs[0].ID)
	if err != nil {
		t.Fatalf("Assignments.ResendNotification: %v", err)
	}
	if !resent.IsSent || resent.DocumentID != doc.ID || resent.SignerID != signerRefs[0].ID {
		t.Errorf("Assignments.ResendNotification returned %+v", resent)
	}

	future := time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02T15:04:05Z")
	if _, err := c.Assignments.ResetExpirationWithRequest(ctx, doc.ID, assignment.ID, models.ResetAssignmentExpirationRequest{ExpiresAt: &future}); err != nil {
		t.Fatalf("Assignments.ResetExpirationWithRequest: %v", err)
	}

	if _, err := c.Assignments.ListWhatsAppNotifications(ctx, doc.ID, assignment.ID); err != nil {
		t.Fatalf("Assignments.ListWhatsAppNotifications: %v", err)
	}
	if err := c.Accounts.Delete(ctx, account.ID, true); err != nil {
		t.Fatalf("Accounts.Delete assignment account: %v", err)
	}
	deleted = true
}
