package resources

import (
	"context"
	"errors"
	"net/http"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
)

func TestStartIdentityVerificationRequiresConsent(t *testing.T) {
	h, _ := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("request sent without biometric-processing consent")
	})
	for _, body := range []*models.StartIdentityVerificationRequest{nil, {Consent: false}} {
		got, err := NewSignerResource(h, "").StartIdentityVerification(context.Background(), "code", body)
		if got != nil || !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Fatalf("result = %v, error = %v; want nil and ErrInvalidInput", got, err)
		}
	}
}

func TestIdentityOperationsReturnAPIErrors(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSONResponse(w, `{"status":404,"message":"Verification not found","data":null}`)
	})
	ctx := context.Background()
	signers, assignments := NewSignerResource(h, ""), NewAssignmentResource(h)
	calls := []func() error{
		func() error { _, err := signers.GetIdentityVerification(ctx, "code"); return err },
		func() error {
			_, err := signers.StartIdentityVerification(ctx, "code", &models.StartIdentityVerificationRequest{Consent: true})
			return err
		},
		func() error { _, err := assignments.GetIdentityReview(ctx, "d", "a", "s"); return err },
		func() error {
			_, err := assignments.DownloadIdentityReviewImage(ctx, "d", "a", "s", "face")
			return err
		},
		func() error {
			_, err := assignments.DecideIdentityReview(ctx, "d", "a", "s", &models.DecideIdentityReviewRequest{Decision: models.IdentityReviewDecisionApproved})
			return err
		},
	}
	for i, call := range calls {
		if err := call(); !sdkerrors.IsStatusCode(err, http.StatusNotFound) {
			t.Errorf("operation %d: %v, want 404 APIError", i, err)
		}
	}
	for _, call := range []func() error{
		func() error { _, err := assignments.GetIdentityReview(ctx, "", "a", "s"); return err },
		func() error { _, err := assignments.DownloadIdentityReviewImage(ctx, "d", "a", "s", ".."); return err },
		func() error { _, err := assignments.DecideIdentityReview(ctx, "d", "", "s", nil); return err },
	} {
		if err := call(); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("invalid path: %v, want ErrInvalidInput", err)
		}
	}
}
