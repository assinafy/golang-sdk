package assinafy

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/assinafy/golang-sdk/models"
)

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
			t.Errorf("cleanup Signers.Delete(%s): %v", signer.ID, err)
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
}

// TestIntegrationTagLifecycle creates a workspace tag, lists it, renames and
// recolors it, then deletes it. Exercises POST/GET/PUT/DELETE on the Tag
// resource against the live API.
func TestIntegrationTagLifecycle(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	name := "go-sdk-audit-" + time.Now().UTC().Format("20060102T150405Z")
	tag, err := c.Tags.Create(ctx, "", &models.CreateTagRequest{Name: name, ClearColor: true})
	if err != nil {
		t.Fatalf("Tags.Create: %v", err)
	}
	t.Cleanup(func() {
		if tag != nil {
			if err := c.Tags.Delete(context.Background(), "", tag.ID, true); err != nil {
				t.Errorf("cleanup Tags.Delete(%s): %v", tag.ID, err)
			}
		}
	})
	if tag.ID == "" || tag.Name != name {
		t.Fatalf("Tags.Create returned %+v", tag)
	}
	if tag.Color != nil {
		t.Errorf("Tags.Create explicit null color returned %v", *tag.Color)
	}

	list, err := c.Tags.List(ctx, "", name)
	if err != nil {
		t.Fatalf("Tags.List: %v", err)
	}
	found := false
	for _, tg := range list {
		if tg.ID == tag.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("created tag %q not returned by List(search=%q): %+v", tag.ID, name, list)
	}

	newName := name + "-renamed"
	newColor := "112233"
	updated, err := c.Tags.Update(ctx, "", tag.ID, &models.UpdateTagRequest{Name: &newName, Color: &newColor})
	if err != nil {
		t.Fatalf("Tags.Update: %v", err)
	}
	if updated.Name != newName || updated.Color == nil || *updated.Color != newColor {
		t.Errorf("Tags.Update returned %+v", updated)
	}
	deleted, err := c.Tags.DeleteWithResult(ctx, "", tag.ID, false)
	if err != nil {
		t.Fatalf("Tags.DeleteWithResult: %v", err)
	}
	if !deleted.Deleted {
		t.Errorf("Tags.DeleteWithResult returned %+v", deleted)
	}
	tag = nil
}

// TestIntegrationDocumentTags uploads a document, replaces and appends tags,
// reads them back, detaches one, and cleans up both the document and the tags
// it auto-created. Exercises the four document-tag endpoints end-to-end.
func TestIntegrationDocumentTags(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	stamp := time.Now().UTC().Format("20060102T150405Z")
	tagA := "go-sdk-audit-a-" + stamp
	tagB := "go-sdk-audit-b-" + stamp

	doc, err := c.Documents.Upload(ctx, "", minimalPDF(), "go-sdk-tags.pdf", nil)
	if err != nil {
		t.Fatalf("Documents.Upload: %v", err)
	}
	t.Cleanup(func() {
		// Detach-created tags survive document deletion; remove them too.
		for _, name := range []string{tagA, tagB} {
			tags, err := c.Tags.List(context.Background(), "", name)
			if err != nil {
				t.Errorf("cleanup Tags.List(%q): %v", name, err)
				continue
			}
			for _, tg := range tags {
				if tg.Name == name {
					if err := c.Tags.Delete(context.Background(), "", tg.ID, true); err != nil {
						t.Errorf("cleanup Tags.Delete(%s): %v", tg.ID, err)
					}
				}
			}
		}
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		for i := 0; i < 6; i++ {
			if err := c.Documents.Delete(cleanCtx, doc.ID); err == nil {
				return
			}
			time.Sleep(2 * time.Second)
		}
		t.Errorf("cleanup: could not delete %s within timeout", doc.ID)
	})

	replaced, err := c.Documents.ReplaceTags(ctx, "", doc.ID, []string{tagA})
	if err != nil {
		t.Fatalf("Documents.ReplaceTags: %v", err)
	}
	if len(replaced) != 1 || replaced[0].Name != tagA {
		t.Errorf("ReplaceTags returned %+v", replaced)
	}

	appended, err := c.Documents.AppendTags(ctx, "", doc.ID, []string{tagB})
	if err != nil {
		t.Fatalf("Documents.AppendTags: %v", err)
	}
	if len(appended) != 2 {
		t.Errorf("AppendTags returned %d tags, want 2: %+v", len(appended), appended)
	}

	listed, err := c.Documents.ListTags(ctx, "", doc.ID)
	if err != nil {
		t.Fatalf("Documents.ListTags: %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("ListTags returned %d tags, want 2: %+v", len(listed), listed)
	}

	detached, err := c.Documents.DetachTagWithResult(ctx, "", doc.ID, listed[0].ID)
	if err != nil {
		t.Fatalf("Documents.DetachTagWithResult: %v", err)
	}
	if !detached.Detached {
		t.Errorf("Documents.DetachTagWithResult returned %+v", detached)
	}
	remaining, err := c.Documents.ListTags(ctx, "", doc.ID)
	if err != nil {
		t.Fatalf("Documents.ListTags (after detach): %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("after detach got %d tags, want 1: %+v", len(remaining), remaining)
	}
}

// TestIntegrationFieldLifecycle creates a custom field definition, reads it
// back, updates it, and deletes it. Exercises POST/GET/PUT/DELETE on the Field
// resource against the live API.
func TestIntegrationFieldLifecycle(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	name := "go-sdk-audit-" + time.Now().UTC().Format("20060102T150405Z")
	required := true
	regex := "/^.+$/"
	field, err := c.Fields.Create(ctx, "", &models.CreateFieldDefinitionRequest{
		Type:       "text",
		Name:       name,
		Regex:      &regex,
		IsRequired: &required,
	})
	if err != nil {
		t.Fatalf("Fields.Create: %v", err)
	}
	t.Cleanup(func() {
		if field != nil {
			if err := c.Fields.Delete(context.Background(), "", field.ID); err != nil {
				t.Errorf("cleanup Fields.Delete(%s): %v", field.ID, err)
			}
		}
	})
	if field.ID == "" || field.Name != name {
		t.Fatalf("Fields.Create returned %+v", field)
	}

	got, err := c.Fields.Get(ctx, "", field.ID)
	if err != nil {
		t.Fatalf("Fields.Get: %v", err)
	}
	if got.ID != field.ID {
		t.Errorf("Fields.Get id = %q, want %q", got.ID, field.ID)
	}
	validated, err := c.Fields.ValidateAuthenticated(ctx, "", field.ID, &models.ValidateFieldRequest{Value: "valid"})
	if err != nil {
		t.Fatalf("Fields.ValidateAuthenticated: %v", err)
	}
	if !validated.Success {
		t.Errorf("Fields.ValidateAuthenticated returned %+v", validated)
	}
	validatedMany, err := c.Fields.ValidateMultipleAuthenticated(ctx, "", []models.ValidateMultipleFieldsRequest{{
		FieldID: field.ID,
		Value:   "valid",
	}})
	if err != nil {
		t.Fatalf("Fields.ValidateMultipleAuthenticated: %v", err)
	}
	if len(validatedMany) != 1 || !validatedMany[0].Success || validatedMany[0].FieldID != field.ID {
		t.Errorf("Fields.ValidateMultipleAuthenticated returned %+v", validatedMany)
	}

	newName := name + "-renamed"
	updated, err := c.Fields.Update(ctx, "", field.ID, &models.UpdateFieldDefinitionRequest{
		Name:       &newName,
		ClearRegex: true,
	})
	if err != nil {
		t.Fatalf("Fields.Update: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Fields.Update name = %q, want %q", updated.Name, newName)
	}
	if updated.Regex != nil {
		t.Errorf("Fields.Update regex = %q, want null", *updated.Regex)
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
