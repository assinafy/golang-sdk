package assinafy

import (
	"context"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
)

// TestIntegrationAccountsList lists the accounts the credential belongs to and
// checks the slim item shape decodes.
func TestIntegrationAccountsList(t *testing.T) {
	c, accountID := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accts, err := c.Accounts.List(ctx)
	if err != nil {
		t.Fatalf("Accounts.List: %v", err)
	}
	found := false
	for _, a := range accts {
		if a.ID == accountID {
			found = true
		}
		if a.ID == "" || a.Name == "" {
			t.Errorf("account item missing id/name: %+v", a)
		}
	}
	if !found {
		t.Errorf("configured account %q not present in Accounts.List result", accountID)
	}
}

// TestIntegrationAccountGet fetches the configured account and checks the
// single-account shape.
func TestIntegrationAccountGet(t *testing.T) {
	c, accountID := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acct, err := c.Accounts.Get(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.Get: %v", err)
	}
	if acct.ID != accountID {
		t.Errorf("account id = %q, want %q", acct.ID, accountID)
	}
	if acct.CreatedAt.IsZero() {
		t.Error("expected created_at to be populated")
	}
}

// TestIntegrationAccountTheme fetches the account branding theme.
func TestIntegrationAccountTheme(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	theme, err := c.Accounts.GetTheme(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.GetTheme: %v", err)
	}
	if theme.AccountName == "" {
		t.Errorf("theme account_name empty: %+v", theme)
	}
}

// TestIntegrationAccountLogoDownload confirms the logo endpoint is reachable. A
// workspace without a logo returns a 404 APIError, which is an acceptable
// outcome; a configured logo returns image bytes.
func TestIntegrationAccountLogoDownload(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	img, err := c.Accounts.DownloadLogo(ctx, "")
	if err != nil {
		if !sdkerrors.IsStatusCode(err, 404) {
			t.Fatalf("Accounts.DownloadLogo: unexpected error: %v", err)
		}
		return // no logo configured — endpoint reachable, 404 is fine
	}
	if len(img) == 0 {
		t.Error("logo download returned no bytes")
	}
}

// TestIntegrationAccountUpdateRoundTrip performs a non-destructive update by
// writing the account's existing name back to it and confirming the response.
func TestIntegrationAccountUpdateRoundTrip(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	current, err := c.Accounts.Get(ctx, "")
	if err != nil {
		t.Fatalf("Accounts.Get: %v", err)
	}
	name := current.Name
	updated, err := c.Accounts.Update(ctx, "", &models.UpdateAccountRequest{Name: &name})
	if err != nil {
		t.Fatalf("Accounts.Update: %v", err)
	}
	if updated.Name != name {
		t.Errorf("Accounts.Update name = %q, want %q", updated.Name, name)
	}
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

	doc := uploadReadyDoc(t, c, ctx)
	original := doc.Name

	renamed, err := c.Documents.Rename(ctx, doc.ID, "go-sdk-audit-renamed.pdf")
	if err != nil {
		t.Fatalf("Documents.Rename: %v", err)
	}
	if renamed.Name != "go-sdk-audit-renamed.pdf" {
		t.Errorf("Rename name = %q, want go-sdk-audit-renamed.pdf", renamed.Name)
	}

	got, err := c.Documents.Get(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Documents.Get after rename: %v", err)
	}
	if got.Name != "go-sdk-audit-renamed.pdf" {
		t.Errorf("after rename, Get name = %q", got.Name)
	}

	if _, err := c.Documents.Rename(ctx, doc.ID, original); err != nil {
		t.Logf("restore original name: %v", err)
	}
}

// TestIntegrationAssignmentsList exercises the account-wide assignments list.
// With API-key auth the sandbox requires a current-account context and returns a
// 400; with a user token it returns a page. Both outcomes prove the request
// encoding and are accepted.
func TestIntegrationAssignmentsList(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.Assignments.List(ctx, &models.ListParams{PerPage: 5})
	if err != nil {
		if !sdkerrors.IsStatusCode(err, 400) {
			t.Fatalf("Assignments.List: unexpected error: %v", err)
		}
		t.Log("Assignments.List returned 400 (expected with API-key auth: needs current-account context)")
		return
	}
	_ = page
}
