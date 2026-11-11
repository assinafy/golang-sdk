package resources

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestAccountSigningFeatures(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/accounts/acc%2F1" || r.URL.Query().Get("expand") != "signing_features" || r.Header.Get("X-Api-Key") != "key" {
			t.Errorf("request = %s %s", r.Method, r.URL)
		}
		writeJSONResponse(w, `{"status":200,"data":{"id":"acc/1","signing_features":{"DigitalCertificate":true,"VerificationIdDocumentFaceMatchBasic":false,"WhatsappNotification":true,"VerificationBypass":false,"Embed":true}}}`)
	})
	params := &models.AccountGetParams{Expand: "signing_features"}
	got, err := NewAccountResource(h, "acc/1").GetWithParams(context.Background(), "", params)
	if err != nil {
		t.Fatal(err)
	}
	want := &models.SigningFeatures{DigitalCertificate: true, WhatsappNotification: true, Embed: true}
	if !reflect.DeepEqual(got.SigningFeatures, want) || params.Expand != "signing_features" {
		t.Fatalf("features = %+v; params = %+v", got.SigningFeatures, params)
	}
}

func TestAccountUsagePreservesUnlimitedBalances(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/accounts/acc%2F1/usage" || r.URL.RawQuery != "" || r.Header.Get("X-Api-Key") != "key" {
			t.Errorf("request = %s %s", r.Method, r.URL)
		}
		writeJSONResponse(w, `{"status":200,"data":{"plan":null,"documents_renewal_date":null,"balances":{"documents":{"balance":null,"limit":null,"spent":3},"credits":{"balance":0,"limit":25.5,"spent":25.5}}}}`)
	})
	got, err := NewAccountResource(h, "acc/1").Usage(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Plan != nil || got.DocumentsRenewalDate != nil || got.Balances.Documents.Balance != nil || got.Balances.Documents.Limit != nil || got.Balances.Documents.Spent != 3 || got.Balances.Credits.Balance == nil || *got.Balances.Credits.Balance != 0 || got.Balances.Credits.Limit == nil || *got.Balances.Credits.Limit != 25.5 || got.Balances.Credits.Spent != 25.5 {
		t.Fatalf("usage = %+v", got)
	}
}

func TestAccountNotificationPolicy(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method || r.URL.EscapedPath() != "/accounts/acc%2F1/notification-policy" || r.Header.Get("X-Api-Key") != "key" {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				if method == http.MethodPut {
					checkJSONContract(t, w, r, map[string]any{"events": map[string]bool{"SignerViewed": false}, "owners_receive_all": false})
				}
				writeJSONResponse(w, `{"status":200,"data":{"editable":true,"owners_receive_all":false,"events":{"SignerViewed":false,"DocumentCompleted":true,"FutureCode":true}}}`)
			})
			r := NewAccountResource(h, "acc/1")
			var got *models.NotificationPolicy
			var err error
			if method == http.MethodGet {
				got, err = r.GetNotificationPolicy(context.Background(), "")
			} else {
				disabled := false
				got, err = r.UpdateNotificationPolicy(context.Background(), "", &models.UpdateNotificationPolicyRequest{Events: models.NotificationPolicyEvents{models.NotificationPreferenceSignerViewed: false}, OwnersReceiveAll: &disabled})
			}
			if err != nil {
				t.Fatal(err)
			}
			if !got.Editable || got.OwnersReceiveAll || got.Events[models.NotificationPreferenceSignerViewed] || !got.Events[models.NotificationPreferenceDocumentCompleted] || !got.Events["FutureCode"] {
				t.Fatalf("policy = %+v", got)
			}
		})
	}
}

func TestAccountFeatureErrors(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSONResponse(w, `{"status":403,"message":"Forbidden"}`)
	})
	ctx := context.Background()
	r := NewAccountResource(h, "account")
	for name, call := range map[string]func(string) error{
		"expanded account": func(id string) error { _, err := r.GetWithParams(ctx, id, nil); return err },
		"usage":            func(id string) error { _, err := r.Usage(ctx, id); return err },
		"get policy":       func(id string) error { _, err := r.GetNotificationPolicy(ctx, id); return err },
		"update policy":    func(id string) error { _, err := r.UpdateNotificationPolicy(ctx, id, nil); return err },
	} {
		if err := call(""); !sdkerrors.IsStatusCode(err, http.StatusForbidden) {
			t.Errorf("%s: %v", name, err)
		}
		if err := call(" "); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("%s blank account: %v", name, err)
		}
	}
}
