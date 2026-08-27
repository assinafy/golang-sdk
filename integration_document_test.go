package assinafy

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/assinafy/golang-sdk/models"
)

// TestIntegrationDocumentDownloads exercises the binary artifact and page
// download endpoints against a freshly processed document.
func TestIntegrationDocumentDownloads(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	doc := uploadReadyDoc(ctx, t, c, "", true)

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

// TestIntegrationDocumentSearch exercises the account document search endpoint.
// An empty result is acceptable; the test only proves the request/response
// encoding works against the live API.
func TestIntegrationDocumentSearch(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Documents.Search(ctx, "", &models.ListParams{PerPage: 5})
	if err != nil {
		t.Fatalf("Documents.Search: %v", err)
	}
	_ = page
}

// TestIntegrationDocumentRename uploads a document, waits until it is renamable,
// renames it, confirms the change, and restores the original name.
func TestIntegrationDocumentRename(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	doc := uploadReadyDoc(ctx, t, c, "", true)
	original := doc.Name

	renamed, err := c.Documents.Rename(ctx, doc.ID, "go-sdk-renamed.pdf")
	if err != nil {
		t.Fatalf("Documents.Rename: %v", err)
	}
	if renamed.Name != "go-sdk-renamed.pdf" {
		t.Errorf("Rename name = %q, want go-sdk-renamed.pdf", renamed.Name)
	}

	got, err := c.Documents.Get(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Documents.Get after rename: %v", err)
	}
	if got.Name != "go-sdk-renamed.pdf" {
		t.Errorf("after rename, Get name = %q", got.Name)
	}

	if _, err := c.Documents.Rename(ctx, doc.ID, original); err != nil {
		t.Errorf("restore original name: %v", err)
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

// TestIntegrationTagLifecycle creates a workspace tag, lists it, renames and
// recolors it, then deletes it. Exercises POST/GET/PUT/DELETE on the Tag
// resource against the live API.
func TestIntegrationTagLifecycle(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	name := "go-sdk-test-" + time.Now().UTC().Format("20060102T150405Z")
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
	tagA := "go-sdk-tag-a-" + stamp
	tagB := "go-sdk-tag-b-" + stamp

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

	name := "go-sdk-test-" + time.Now().UTC().Format("20060102T150405Z")
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
