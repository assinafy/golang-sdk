package assinafy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

// TestIntegrationContractLifecycle tracks terms on temporary unsigned documents,
// links and unlinks an instrument, renews and terminates the root, then deletes
// both documents. It does not request signatures or change workspace settings.
func TestIntegrationContractLifecycle(t *testing.T) {
	c, _ := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	doc := uploadReadyDoc(ctx, t, c, "", true)
	if _, err := c.Contracts.Get(ctx, "", doc.ID); !sdkerrors.IsStatusCode(err, 404) {
		t.Fatalf("Get before terms: %v", err)
	}
	status := models.ContractStatusActive
	endDate := time.Now().UTC().AddDate(1, 0, 0).Format(time.DateOnly)
	initial, err := c.Contracts.Set(ctx, "", doc.ID, &models.SetDocumentContractRequest{
		Status: &status, CounterpartyName: models.NullableValue("SDK Example Company"),
		Value: models.NullableValue(json.Number("12.34")), Currency: models.NullableValue("BRL"),
		EndDate: models.NullableValue(endDate), AutoRenew: models.NullableValue(false),
		NoticePeriodDays:   models.NullableValue(0),
		ExternalReferences: models.NullableValue([]models.ContractExternalReference{{System: "go-sdk-test", ID: doc.ID}}),
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if initial.Contract == nil || initial.Contract.DocumentID != doc.ID || initial.Contract.Value == nil || initial.Contract.Value.String() != "12.34" {
		t.Fatal("Set did not return the saved terms")
	}
	read, err := c.Contracts.Get(ctx, "", doc.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if read.Contract == nil || read.Contract.UpdatedAt == "" {
		t.Fatal("Get omitted contract timestamp")
	}

	page, err := c.Contracts.List(ctx, "", &models.ContractListParams{
		ExternalSystem: "go-sdk-test", ExternalID: doc.ID, Expand: "document",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].DocumentID != doc.ID || page.Data[0].Document == nil || page.Pagination.TotalCount != 1 {
		t.Fatal("List did not retain the filtered contract, expansion and pagination")
	}

	stale := models.Timestamp("2000-01-01T00:00:00Z")
	_, err = c.Contracts.Set(ctx, "", doc.ID, &models.SetDocumentContractRequest{UpdatedAt: &stale, Value: models.NullableValue(json.Number("99"))})
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict || apiErr.Code != "ContractChanged" {
		t.Fatalf("stale write: %v", err)
	}
	_, err = c.Contracts.Set(ctx, "", doc.ID, &models.SetDocumentContractRequest{Value: models.NullableValue(json.Number("-1"))})
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest || len(apiErr.Errors) == 0 || apiErr.Errors[0].Code != "TooSmall" {
		t.Fatalf("invalid value: %v", err)
	}

	cleared, err := c.Contracts.Set(ctx, "", doc.ID, &models.SetDocumentContractRequest{
		Value: &models.Nullable[json.Number]{}, NoticePeriodDays: &models.Nullable[int]{},
		AutoRenew: models.NullableValue(false), UpdatedAt: &read.Contract.UpdatedAt,
		ExternalReferences: models.NullableValue([]models.ContractExternalReference{}),
	})
	if err != nil {
		t.Fatalf("clear terms: %v", err)
	}
	if cleared.Contract == nil || cleared.Contract.Value != nil || cleared.Contract.NoticePeriodDays != nil || cleared.Contract.AutoRenew || len(cleared.Contract.ExternalReferences) != 0 {
		t.Fatal("nullable update did not clear terms")
	}

	instrument := uploadReadyDoc(ctx, t, c, "", true)
	relation := models.ContractRelationAmendment
	linked, err := c.Contracts.Set(ctx, "", instrument.ID, &models.SetDocumentContractRequest{
		ParentDocumentID: models.NullableValue(doc.ID), RelationType: &relation,
		Value: models.NullableValue(json.Number("20.50")),
	})
	if err != nil {
		t.Fatalf("link instrument: %v", err)
	}
	if linked.Amendment == nil || linked.Amendment.Status != "pending" || linked.Amendment.ParentDocumentID != doc.ID {
		t.Fatal("link did not return a pending amendment")
	}
	family, err := c.Contracts.Related(ctx, "", instrument.ID)
	if err != nil {
		t.Fatalf("Related: %v", err)
	}
	if family.DocumentID != doc.ID || len(family.Amendments) != 1 || family.Value != nil {
		t.Fatal("pending amendment changed parent or was absent from family")
	}
	if _, err := c.Contracts.Set(ctx, "", instrument.ID, &models.SetDocumentContractRequest{ParentDocumentID: &models.Nullable[string]{}}); err != nil {
		t.Fatalf("unlink instrument: %v", err)
	}
	if _, err := c.Contracts.Get(ctx, "", instrument.ID); !sdkerrors.IsStatusCode(err, 404) {
		t.Fatalf("Get after unlink: %v", err)
	}
	family, err = c.Contracts.Related(ctx, "", doc.ID)
	if err != nil {
		t.Fatalf("Related after unlink: %v", err)
	}
	if len(family.Amendments) != 0 {
		t.Fatal("unlinked instrument remains in the family")
	}

	renewed, err := c.Contracts.Renew(ctx, "", doc.ID, &models.RenewContractRequest{Term: &family.Term, UpdatedAt: &family.UpdatedAt})
	if err != nil {
		t.Fatalf("Renew: %v", err)
	}
	if renewed.Term != family.Term+1 || renewed.EndDate == nil || *renewed.EndDate <= endDate {
		t.Fatal("renewal did not advance the term and end date")
	}
	if _, err := c.Contracts.Renew(ctx, "", doc.ID, &models.RenewContractRequest{Term: &family.Term}); !sdkerrors.IsStatusCode(err, 400) {
		t.Fatalf("duplicate renewal: %v", err)
	}
	reason := "SDK temporary contract completed"
	terminated, err := c.Contracts.Terminate(ctx, "", doc.ID, &models.TerminateContractRequest{Reason: &reason, UpdatedAt: &renewed.UpdatedAt})
	if err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	if terminated.Status != models.ContractStatusTerminated {
		t.Fatalf("termination status = %s", terminated.Status)
	}
	activities, err := c.Documents.ActivitiesWithParams(ctx, doc.ID, &models.ListParams{PerPage: 100})
	if err != nil {
		t.Fatalf("contract activities: %v", err)
	}
	found := false
	for _, activity := range activities.Data {
		if activity.Event == "document_contract_terminated" {
			found = true
		}
	}
	if !found {
		t.Fatal("termination event was not recorded")
	}
}
