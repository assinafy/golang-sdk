package assinafy

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

// TestIntegrationAssignmentLifecycle exercises virtual and collect assignment
// management in a disposable account: signers, field placements, cost estimates,
// notifications, and expiration. It SENDS real
// signature-request emails, so it is opt-in via ASSINAFY_RUN_ASSIGNMENT_TESTS=1.
func TestIntegrationAssignmentLifecycle(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_ASSIGNMENT_TESTS") != "1" {
		t.Skip("set ASSINAFY_RUN_ASSIGNMENT_TESTS=1 to run the assignment flow (sends emails)")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	account, err := c.Accounts.Create(ctx, &models.CreateAccountRequest{
		Name: "go-sdk-assignment-" + time.Now().UTC().Format("20060102T150405.000000000Z"),
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

	doc := uploadReadyDoc(ctx, t, c, account.ID, false)

	emails := []string{
		strings.TrimSpace(os.Getenv("ASSINAFY_TEST_EMAIL_PRIMARY")),
		strings.TrimSpace(os.Getenv("ASSINAFY_TEST_EMAIL_SECONDARY")),
	}
	if emails[0] == "" || emails[1] == "" {
		t.Fatal("ASSINAFY_TEST_EMAIL_PRIMARY and ASSINAFY_TEST_EMAIL_SECONDARY are required for assignment tests")
	}
	signerRefs := make([]models.SignerReference, 0, len(emails))
	for i, email := range emails {
		signer, err := c.Signers.Create(ctx, account.ID, &models.CreateSignerRequest{
			FullName: "Go SDK Sandbox Signer",
			Email:    &email,
		})
		if err != nil {
			t.Fatalf("Signers.Create recipient %d: %v", i+1, err)
		}
		signerRefs = append(signerRefs, models.SignerReference{
			ID:                  signer.ID,
			VerificationMethod:  "Email",
			NotificationMethods: []string{"Email"},
		})
	}

	message := "Assinafy Go SDK sandbox test — no action is required."
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
	sent, err := c.PublicDocuments.SendToken(ctx, doc.ID, &models.SendDocumentTokenRequest{Recipient: emails[1]})
	if err != nil {
		t.Fatalf("PublicDocuments.SendToken: %v", err)
	}
	if sent.Channel != models.SendTokenChannelEmail || sent.Recipient != emails[1] || sent.Document.ID != doc.ID || sent.Document.PageCount == "" {
		t.Errorf("PublicDocuments.SendToken = %+v", sent)
	}

	workspaceDocs, err := c.SignerDocuments.ListAsWorkspace(ctx, signerRefs[0].ID, nil)
	if err != nil {
		t.Fatalf("SignerDocuments.ListAsWorkspace: %v", err)
	}
	if len(workspaceDocs.Data) == 0 || workspaceDocs.Pagination.TotalCount == 0 {
		t.Errorf("SignerDocuments.ListAsWorkspace = %+v", workspaceDocs)
	}
	if _, err := c.SignerDocuments.SearchAsWorkspace(ctx, signerRefs[0].ID, "go-sdk"); err != nil {
		t.Fatalf("SignerDocuments.SearchAsWorkspace: %v", err)
	}
	if _, err := c.SignerDocuments.List(ctx, signerRefs[0].ID, "", nil); !errors.Is(err, sdkerrors.ErrInvalidInput) {
		t.Errorf("SignerDocuments.List with an empty code = %v, want ErrInvalidInput", err)
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
	extended, err := c.Assignments.ResetExpirationWithRequest(ctx, doc.ID, assignment.ID, models.ResetAssignmentExpirationRequest{ExpiresAt: &future})
	if err != nil {
		t.Fatalf("Assignments.ResetExpirationWithRequest: %v", err)
	}
	if extended.ExpiresAt == nil {
		t.Error("ResetExpirationWithRequest left expires_at empty")
	}
	cleared, err := c.Assignments.ResetExpiration(ctx, doc.ID, assignment.ID, nil)
	if err != nil {
		t.Fatalf("Assignments.ResetExpiration(nil): %v", err)
	}
	if cleared.ExpiresAt != nil {
		t.Errorf("ResetExpiration(nil) expires_at = %q, want null", *cleared.ExpiresAt)
	}

	if _, err := c.Assignments.ListWhatsAppNotifications(ctx, doc.ID, assignment.ID); err != nil {
		t.Fatalf("Assignments.ListWhatsAppNotifications: %v", err)
	}
	collectDoc := uploadReadyDoc(ctx, t, c, account.ID, false)
	if len(collectDoc.Pages) == 0 {
		t.Fatal("metadata-ready document has no pages for collect fields")
	}
	field, err := c.Fields.Create(ctx, account.ID, &models.CreateFieldDefinitionRequest{Type: "text", Name: "Approval"})
	if err != nil {
		t.Fatalf("Fields.Create for collect: %v", err)
	}
	entry := models.AssignmentEntry{PageID: collectDoc.Pages[0].ID, Fields: []models.AssignmentField{{
		SignerID: signerRefs[0].ID, FieldID: field.ID,
		DisplaySettings: models.DisplaySettings{Left: 20, Top: 20, Width: 200, Height: 30, FontSize: 12},
	}}}
	collectEstimate, err := c.Assignments.EstimateCostWithRequest(ctx, collectDoc.ID, &models.EstimateAssignmentCostRequest{
		Method: models.MethodCollect, Signers: []models.EstimateAssignmentCostSigner{{}}, Entries: []any{entry},
	})
	if err != nil {
		t.Fatalf("Assignments.EstimateCostWithRequest for collect: %v", err)
	}
	if !collectEstimate.HasSufficientResources {
		t.Fatal("collect estimate has insufficient resources")
	}
	collect, err := c.Assignments.Create(ctx, collectDoc.ID, &models.CreateAssignmentRequest{
		Method: models.MethodCollect, Signers: signerRefs[:1], Entries: []models.AssignmentEntry{entry},
	})
	if err != nil {
		t.Fatalf("Assignments.Create for collect: %v", err)
	}
	if collect.Method != models.MethodCollect || len(collect.Items) != 1 || collect.Items[0].Page == nil || collect.Items[0].Page.ID != entry.PageID {
		t.Error("collect assignment did not retain the placed page field")
	}
	if err := c.Accounts.Delete(ctx, account.ID, true); err != nil {
		t.Fatalf("Accounts.Delete assignment account: %v", err)
	}
	deleted = true
}

// TestIntegrationAssignmentsList exercises the account-wide assignments list.
// A positive response requires authenticated user current-account context.
func TestIntegrationAssignmentsList(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Assignments.List(ctx, &models.ListParams{PerPage: 5})
	if err != nil {
		if !sdkerrors.IsStatusCode(err, 400) || !strings.Contains(err.Error(), "contexto de conta") {
			t.Fatalf("Assignments.List: unexpected error: %v", err)
		}
		t.Skipf("positive Assignments.List response requires a user token: %v", err)
	}
	_ = page
}

func TestIntegrationListSigners(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Signers.List(ctx, "", &models.ListParams{PerPage: 5})
	if err != nil {
		t.Fatalf("Signers.List: %v", err)
	}
	_ = page
}

// TestIntegrationSignerLifecycle creates a signer, reads it back, updates it,
// and deletes it. Touches POST/GET/PUT/DELETE on the signer resource to prove
// the path encoding and JSON body shapes match the API.
func TestIntegrationSignerLifecycle(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	email := "go-sdk-test-" + time.Now().UTC().Format("20060102T150405Z") + "@example.test"
	signer, err := c.Signers.Create(ctx, "", &models.CreateSignerRequest{
		FullName: "Go SDK Test",
		Email:    &email,
	})
	if err != nil {
		t.Fatalf("Signers.Create: %v", err)
	}
	t.Cleanup(func() {
		if signer == nil {
			return
		}
		if err := c.Signers.Delete(context.Background(), "", signer.ID); err != nil {
			t.Errorf("cleanup Signers.Delete(%s): %v", signer.ID, err)
		}
	})

	got, err := c.Signers.Get(ctx, "", signer.ID)
	if err != nil {
		t.Fatalf("Signers.Get: %v", err)
	}
	if got.ID != signer.ID || got.FullName != "Go SDK Test" {
		t.Errorf("Signers.Get mismatch: %+v", got)
	}

	newName := "Go SDK Test Renamed"
	updated, err := c.Signers.Update(ctx, "", signer.ID, &models.UpdateSignerRequest{FullName: &newName})
	if err != nil {
		t.Fatalf("Signers.Update: %v", err)
	}
	if updated.FullName != newName {
		t.Errorf("Signers.Update full_name = %q, want %q", updated.FullName, newName)
	}
}

// TestIntegrationDocumentUploadAndEstimateCost uploads a minimal PDF, calls
// the assignment estimate-cost endpoint, and deletes the document. This
// proves the multipart upload, the cost-estimate response decoding, and the
// document delete flow end-to-end.
func TestIntegrationDocumentUploadAndEstimateCost(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	doc, err := c.Documents.Upload(ctx, "", minimalPDF(), "go-sdk-test.pdf", nil)
	if err != nil {
		t.Fatalf("Documents.Upload: %v", err)
	}
	t.Cleanup(func() {
		if doc == nil {
			return
		}
		// The API blocks deletion until metadata_processing finishes. Poll a
		// few times before giving up so the workspace stays clean.
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		for i := 0; i < 6; i++ {
			err := c.Documents.Delete(cleanCtx, doc.ID)
			if err == nil {
				return
			}
			time.Sleep(2 * time.Second)
		}
		t.Errorf("cleanup: could not delete %s within timeout (still processing)", doc.ID)
	})
	if doc.ID == "" {
		t.Fatal("expected non-empty document id")
	}

	estimate, err := c.Assignments.EstimateCostWithRequest(ctx, doc.ID, &models.EstimateAssignmentCostRequest{
		Method:  models.MethodVirtual,
		Signers: []models.EstimateAssignmentCostSigner{{}},
	})
	if err != nil {
		t.Fatalf("Assignments.EstimateCostWithRequest: %v", err)
	}
	if estimate.Documents != 1 {
		t.Errorf("estimate.Documents = %v, want 1", estimate.Documents)
	}

	// WhatsApp and certificate verification are priced even when the plan
	// cannot use them; a plan gate answers 403.
	for _, signer := range []models.EstimateAssignmentCostSigner{
		{VerificationMethod: models.VerificationMethodWhatsApp, NotificationMethods: []string{models.NotificationMethodWhatsApp}},
		{VerificationMethod: models.VerificationMethodDigitalCertificate, NotificationMethods: []string{models.NotificationMethodEmail}},
	} {
		estimate, err := c.Assignments.EstimateCostWithRequest(ctx, doc.ID, &models.EstimateAssignmentCostRequest{
			Method:  models.MethodVirtual,
			Signers: []models.EstimateAssignmentCostSigner{signer},
		})
		switch {
		case sdkerrors.IsStatusCode(err, 403):
			t.Logf("%s estimate is not available on this plan: %v", signer.VerificationMethod, err)
		case err != nil:
			t.Errorf("EstimateCostWithRequest(%s): %v", signer.VerificationMethod, err)
		case estimate.TotalCredits <= 0:
			t.Errorf("EstimateCostWithRequest(%s) total_credits = %v, want > 0", signer.VerificationMethod, estimate.TotalCredits)
		}
	}
}
