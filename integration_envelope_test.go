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

// TestIntegrationEnvelopeLifecycle sends email invitations for disposable
// documents and exercises envelope management with a zero-credit ceiling.
func TestIntegrationEnvelopeLifecycle(t *testing.T) {
	if os.Getenv("ASSINAFY_RUN_ASSIGNMENT_TESTS") != "1" {
		t.Skip("set ASSINAFY_RUN_ASSIGNMENT_TESTS=1 to run the envelope flow (sends emails)")
	}
	if !isProductionBaseURL(os.Getenv("ASSINAFY_BASE_URL")) {
		t.Skip("envelope lifecycle requires the production API")
	}
	email := strings.TrimSpace(os.Getenv("ASSINAFY_TEST_EMAIL_PRIMARY"))
	if email == "" {
		t.Fatal("ASSINAFY_TEST_EMAIL_PRIMARY is required for envelope notifications")
	}
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	signer := integrationEmailSigner(ctx, t, c, email)

	first := uploadReadyDoc(ctx, t, c, "", true)
	second := uploadReadyDoc(ctx, t, c, "", true)
	third := uploadReadyDoc(ctx, t, c, "", true)
	name := "go-sdk-envelope-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	message := "Assinafy Go SDK integration test. No signature is required."
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
		t.Skipf("envelope requires resources outside the zero-credit test: credits=%v, sufficient=%v", estimate.TotalCredits, estimate.HasSufficientResources)
	}
	envelope, err := c.Envelopes.Create(ctx, "", body)
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict && apiErr.Code == "CreditCeilingExceeded" {
		t.Skip("envelope cost exceeded the zero-credit ceiling")
	}
	if err != nil {
		t.Fatalf("Envelopes.Create: %v", err)
	}
	deleted := false
	t.Cleanup(func() {
		if deleted {
			return
		}
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanCancel()
		if _, err := c.Envelopes.Cancel(cleanCtx, "", envelope.ID); err != nil {
			t.Errorf("cleanup Envelopes.Cancel: %v", err)
			return
		}
		if err := c.Envelopes.Delete(cleanCtx, "", envelope.ID); err != nil {
			t.Errorf("cleanup Envelopes.Delete: %v", err)
		}
	})
	if envelope.ID == "" || len(envelope.Documents) != 2 || envelope.Name != name {
		t.Fatal("created envelope omitted its identifier, name or documents")
	}
	replayed, err := c.Envelopes.Create(ctx, "", body)
	if err != nil || replayed.ID != envelope.ID {
		t.Fatalf("Envelopes.Create idempotent replay: %v", err)
	}
	read, err := c.Envelopes.Get(ctx, "", envelope.ID)
	if err != nil || read.ID != envelope.ID || len(read.Documents) != 2 {
		t.Fatalf("Envelopes.Get: %v", err)
	}
	listed, err := c.Envelopes.List(ctx, "", &models.ListParams{Search: name, Status: "open", PerPage: 20})
	if err != nil {
		t.Fatalf("Envelopes.List: %v", err)
	}
	found := false
	for _, item := range listed.Data {
		found = found || item.ID == envelope.ID
	}
	if !found {
		t.Fatal("created envelope missing from the filtered list")
	}
	updatedName := name + "-updated"
	updated, err := c.Envelopes.Update(ctx, "", envelope.ID, &models.UpdateEnvelopeRequest{
		Name: &updatedName, Message: &models.Nullable[string]{}, DocumentIDs: []string{second.ID, first.ID},
	})
	if err != nil {
		t.Fatalf("Envelopes.Update: %v", err)
	}
	if updated.Name != updatedName || updated.Message != nil || len(updated.Documents) != 2 || updated.Documents[0].ID != second.ID {
		t.Fatal("envelope update did not retain the name, cleared message and document order")
	}
	archive, err := c.Envelopes.Download(ctx, "", envelope.ID, "original")
	if err != nil {
		t.Fatalf("Envelopes.Download: %v", err)
	}
	zipFile, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil || len(zipFile.File) != 2 {
		t.Fatalf("envelope original ZIP must contain two documents: %v", err)
	}
	resent, err := c.Envelopes.ResendNotification(ctx, "", envelope.ID, signer.ID)
	if err != nil {
		t.Fatalf("Envelopes.ResendNotification: %v", err)
	}
	if !resent.IsSent || resent.EnvelopeID != envelope.ID || resent.SignerID != signer.ID {
		t.Fatal("envelope reminder was not sent to the selected signer")
	}
	added, err := c.Envelopes.AddDocument(ctx, "", envelope.ID, &models.AddEnvelopeDocumentRequest{
		EnvelopeDocumentRequest: models.EnvelopeDocumentRequest{ID: third.ID, Method: models.MethodVirtual, SignerIDs: []string{signer.ID}}, MaxTotalCredits: &zero,
	})
	if err != nil {
		t.Fatalf("Envelopes.AddDocument: %v", err)
	}
	if len(added.Documents) != 3 || added.Documents[2].ID != third.ID {
		t.Fatal("added document is not last in the envelope")
	}
	removed, err := c.Envelopes.RemoveDocument(ctx, "", envelope.ID, third.ID)
	if err != nil || len(removed.Documents) != 2 {
		t.Fatalf("Envelopes.RemoveDocument: %v", err)
	}
	activities, err := c.Envelopes.Activities(ctx, "", envelope.ID, &models.ListParams{Page: 1, PerPage: 10, Sort: "id"})
	if err != nil {
		t.Fatalf("Envelopes.Activities: %v", err)
	}
	if len(activities.Data) == 0 || activities.Pagination.TotalCount == 0 || activities.Data[0].DocumentID == "" {
		t.Fatal("envelope activities omitted document references or pagination")
	}
	if err := c.Envelopes.Delete(ctx, "", envelope.ID); !sdkerrors.IsStatusCode(err, 409) {
		t.Fatalf("deleting an open envelope: %v, want 409", err)
	}
	cancelled, err := c.Envelopes.Cancel(ctx, "", envelope.ID)
	if err != nil || len(cancelled) != 2 {
		t.Fatalf("Envelopes.Cancel: %v", err)
	}
	for _, item := range cancelled {
		if !item.Cancelled || item.Status != models.StatusRejectedByUser {
			t.Error("envelope cancellation did not cancel a pending document")
		}
	}
	if err := c.Envelopes.Delete(ctx, "", envelope.ID); err != nil {
		t.Fatalf("Envelopes.Delete: %v", err)
	}
	deleted = true
	if _, err := c.Envelopes.Get(ctx, "", envelope.ID); !sdkerrors.IsStatusCode(err, 404) {
		t.Fatalf("deleted envelope read: %v, want 404", err)
	}
}

func integrationEmailSigner(ctx context.Context, t *testing.T, c *Client, email string) *models.Signer {
	t.Helper()
	// An existing contact is reused without changing or deleting it.
	page, err := c.Signers.List(ctx, "", &models.ListParams{Search: email})
	if err != nil {
		t.Fatalf("Signers.List: %v", err)
	}
	for i := range page.Data {
		if page.Data[i].Email != nil && strings.EqualFold(*page.Data[i].Email, email) {
			return &page.Data[i]
		}
	}
	signer, err := c.Signers.Create(ctx, "", &models.CreateSignerRequest{FullName: "Go SDK Envelope Test", Email: &email})
	if err != nil {
		t.Fatalf("Signers.Create: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		documents, err := c.SignerDocuments.ListAsWorkspace(cleanCtx, signer.ID, &models.ListParams{PerPage: 1})
		if err != nil {
			t.Errorf("cleanup read new signer's documents: %v", err)
			return
		}
		if len(documents.Data) != 0 {
			if documents.Data[0].Status == models.StatusCertificated {
				t.Logf("Retaining newly created signer %s associated with certified document %s", signer.ID, documents.Data[0].ID)
			} else {
				t.Errorf("cleanup new signer %s still has unsigned document %s in status %s", signer.ID, documents.Data[0].ID, documents.Data[0].Status)
			}
			return
		}
		if err := c.Signers.Delete(cleanCtx, "", signer.ID); err != nil {
			t.Errorf("cleanup envelope signer: %v", err)
		}
	})
	return signer
}
