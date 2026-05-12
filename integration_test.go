package assinafy

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/assinafy/golang-sdk/models"
)

// integrationClient builds a client from environment variables. The test is
// skipped when the credentials are not present so unit-only CI runs pass.
func integrationClient(t *testing.T) (*Client, string) {
	t.Helper()
	apiKey := os.Getenv("ASSINAFY_API_KEY")
	accountID := os.Getenv("ASSINAFY_ACCOUNT_ID")
	if apiKey == "" || accountID == "" {
		t.Skip("ASSINAFY_API_KEY and ASSINAFY_ACCOUNT_ID must be set for integration tests")
	}
	c, err := NewClient(ClientOptions{APIKey: apiKey, AccountID: accountID, Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, accountID
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

func TestIntegrationListDocuments(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Documents.List(ctx, "", &models.ListParams{PerPage: 5})
	if err != nil {
		t.Fatalf("Documents.List: %v", err)
	}
	_ = page // Empty result is allowed; we only care that the call succeeds.
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

func TestIntegrationListTemplates(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Templates.List(ctx, "", &models.ListParams{PerPage: 5})
	if err != nil {
		t.Fatalf("Templates.List: %v", err)
	}
	_ = page
}

func TestIntegrationListFields(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fields, err := c.Fields.List(ctx, "", &models.ListFieldDefinitionsParams{IncludeStandard: true})
	if err != nil {
		t.Fatalf("Fields.List: %v", err)
	}
	if len(fields) == 0 {
		t.Fatal("expected at least one field definition")
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

// TestIntegrationSignerLifecycle creates a signer, reads it back, updates it,
// and deletes it. Touches POST/GET/PUT/DELETE on the signer resource to prove
// the path encoding and JSON body shapes match the API.
func TestIntegrationSignerLifecycle(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	email := "go-sdk-audit-" + time.Now().UTC().Format("20060102T150405Z") + "@example.test"
	signer, err := c.Signers.Create(ctx, "", &models.CreateSignerRequest{
		FullName: "Go SDK Audit",
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
			t.Logf("cleanup Signers.Delete(%s): %v", signer.ID, err)
		}
	})

	got, err := c.Signers.Get(ctx, "", signer.ID)
	if err != nil {
		t.Fatalf("Signers.Get: %v", err)
	}
	if got.ID != signer.ID || got.FullName != "Go SDK Audit" {
		t.Errorf("Signers.Get mismatch: %+v", got)
	}

	newName := "Go SDK Audit Renamed"
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

	doc, err := c.Documents.Upload(ctx, "", minimalPDF(), "go-sdk-audit.pdf", nil)
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
		t.Logf("cleanup: could not delete %s within timeout (still processing)", doc.ID)
	})
	if doc.ID == "" {
		t.Fatal("expected non-empty document id")
	}

	estimate, err := c.Assignments.EstimateCost(ctx, doc.ID, &models.CreateAssignmentRequest{
		Method:  models.MethodVirtual,
		Signers: []models.SignerReference{{}},
	})
	if err != nil {
		t.Fatalf("Assignments.EstimateCost: %v", err)
	}
	if estimate.Documents != 1 {
		t.Errorf("estimate.Documents = %v, want 1", estimate.Documents)
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

// TestIntegrationGetDocumentAndActivities exercises the document detail and
// activities endpoints when at least one document exists. Activities are the
// trickiest payload shape (payload can be `[]` when empty, an object when
// populated), so this is the canary for upstream JSON changes.
func TestIntegrationGetDocumentAndActivities(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Documents.List(ctx, "", &models.ListParams{PerPage: 1})
	if err != nil {
		t.Fatalf("Documents.List: %v", err)
	}
	if len(page.Data) == 0 {
		t.Skip("workspace has no documents to exercise Get/Activities")
	}
	docID := page.Data[0].ID

	doc, err := c.Documents.Get(ctx, docID)
	if err != nil {
		t.Fatalf("Documents.Get(%s): %v", docID, err)
	}
	if doc.ID != docID {
		t.Errorf("returned doc id %q, want %q", doc.ID, docID)
	}

	acts, err := c.Documents.Activities(ctx, docID)
	if err != nil {
		t.Fatalf("Documents.Activities(%s): %v", docID, err)
	}
	_ = acts
}
