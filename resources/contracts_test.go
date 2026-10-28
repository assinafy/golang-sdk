package resources

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

const contractResponse = `{"status":200,"data":{"resource":"document_contract","document_id":"doc","status":"active","value":123.45,"auto_renew":false,"term":1,"external_references":[],"updated_at":"2026-10-28T12:00:00Z","amendments":[{"document_id":"instrument","relation_type":"renewal","status":"pending","number":null,"terms":{"end_date":"2028-01-01"},"changes":{"term":{"from":1,"to":2}}}]}}`

func TestContractsList(t *testing.T) {
	zero := 0
	for _, params := range []*models.ContractListParams{nil, {
		ListParams:   models.ListParams{Page: 2, PerPage: 10, Search: "service", Status: "active,expired", Sort: "end_date"},
		Counterparty: "Example", EndDateFrom: "2026-01-01", EndDateTo: "2027-01-01", EndingWithinDays: &zero,
		ExternalSystem: "crm", ExternalID: "ref/1", Expand: "document",
	}} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/accounts/default/contracts" || r.Header.Get("X-Api-Key") != "key" {
				t.Errorf("request = %s %s", r.Method, r.URL)
			}
			want := url.Values{"page": {"1"}, "per-page": {"25"}}
			if params != nil {
				want = url.Values{"page": {"2"}, "per-page": {"10"}, "search": {"service"}, "status": {"active,expired"}, "sort": {"end_date"}, "counterparty": {"Example"}, "end_date_from": {"2026-01-01"}, "end_date_to": {"2027-01-01"}, "ending_within_days": {"0"}, "external_system": {"crm"}, "external_id": {"ref/1"}, "expand": {"document"}}
			}
			if !reflect.DeepEqual(r.URL.Query(), want) {
				t.Errorf("query = %v, want %v", r.URL.Query(), want)
			}
			w.Header().Set("X-Pagination-Total-Count", "8")
			w.Header().Set("X-Pagination-Current-Page", "2")
			writeJSONResponse(w, `{"status":200,"data":[{"resource":"document_contract","document_id":"doc","value":123.45,"term":1,"document":{"id":"doc","has_contract":true}}]}`)
		})
		got, err := NewContractResource(h, "default").List(context.Background(), "", params)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Data) != 1 || got.Data[0].DocumentID != "doc" || got.Data[0].Value.String() != "123.45" || got.Data[0].Document == nil || got.Pagination.TotalCount != 8 || got.Pagination.CurrentPage != 2 {
			t.Fatalf("result = %+v", got)
		}
	}
}

func TestContractsWire(t *testing.T) {
	status := models.ContractStatusActive
	relation := models.ContractRelationAmendment
	stamp := models.Timestamp("2026-10-28T12:00:00Z")
	term := 1
	reason := "End of service"
	ctx := context.Background()
	for _, tc := range []struct {
		name, method, suffix, body string
		call                       func(*ContractResource) (*models.DocumentContract, error)
	}{
		{"get", "GET", "", "", func(r *ContractResource) (*models.DocumentContract, error) {
			got, err := r.Get(ctx, "account/1", "doc/1")
			if got == nil {
				return nil, err
			}
			return got.Contract, err
		}},
		{"set", "PUT", "", `{"status":"active","counterparty_name":"Example","counterparty_taxpayer_id":null,"value":123.45,"currency":"BRL","start_date":"2026-01-01","end_date":"2027-01-01","auto_renew":false,"renewal_term_months":12,"notice_period_days":0,"owner_id":null,"metadata":{"crm":"won"},"external_references":[{"system":"crm","id":"record"}],"parent_document_id":null,"relation_type":"amendment","updated_at":"2026-10-28T12:00:00Z"}`, func(r *ContractResource) (*models.DocumentContract, error) {
			got, err := r.Set(ctx, "account/1", "doc/1", &models.SetDocumentContractRequest{
				Status: &status, CounterpartyName: models.NullableValue("Example"), CounterpartyTaxpayerID: &models.Nullable[string]{},
				Value: models.NullableValue(json.Number("123.45")), Currency: models.NullableValue("BRL"), StartDate: models.NullableValue("2026-01-01"), EndDate: models.NullableValue("2027-01-01"), AutoRenew: models.NullableValue(false), RenewalTermMonths: models.NullableValue(12), NoticePeriodDays: models.NullableValue(0), OwnerID: &models.Nullable[string]{}, Metadata: models.NullableValue(map[string]string{"crm": "won"}), ExternalReferences: models.NullableValue([]models.ContractExternalReference{{System: "crm", ID: "record"}}), ParentDocumentID: &models.Nullable[string]{}, RelationType: &relation, UpdatedAt: &stamp,
			})
			if got == nil {
				return nil, err
			}
			return got.Contract, err
		}},
		{"related", "GET", "/related", "", func(r *ContractResource) (*models.DocumentContract, error) {
			return r.Related(ctx, "account/1", "doc/1")
		}},
		{"renew", "POST", "/renew", `{"end_date":"2028-01-01","value":null,"term":1,"updated_at":"2026-10-28T12:00:00Z"}`, func(r *ContractResource) (*models.DocumentContract, error) {
			return r.Renew(ctx, "account/1", "doc/1", &models.RenewContractRequest{EndDate: models.NullableValue("2028-01-01"), Value: &models.Nullable[json.Number]{}, Term: &term, UpdatedAt: &stamp})
		}},
		{"terminate", "POST", "/terminate", `{"reason":"End of service","updated_at":"2026-10-28T12:00:00Z"}`, func(r *ContractResource) (*models.DocumentContract, error) {
			return r.Terminate(ctx, "account/1", "doc/1", &models.TerminateContractRequest{Reason: &reason, UpdatedAt: &stamp})
		}},
		{"terminate empty", "POST", "/terminate", `{}`, func(r *ContractResource) (*models.DocumentContract, error) {
			return r.Terminate(ctx, "account/1", "doc/1", nil)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.EscapedPath() != "/accounts/account%2F1/documents/doc%2F1/contract"+tc.suffix || r.URL.RawQuery != "" || r.Header.Get("X-Api-Key") != "key" {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				if tc.body != "" {
					var want any
					if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
						t.Fatal(err)
					}
					if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
						t.Errorf("body = %#v, want %#v", got, want)
					}
				}
				writeJSONResponse(w, contractResponse)
			})
			got, err := tc.call(NewContractResource(h, "default"))
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || got.DocumentID != "doc" || got.Value.String() != "123.45" || len(got.Amendments) != 1 || string(got.Amendments[0].Changes["term"].To) != "2" {
				t.Fatalf("result = %+v", got)
			}
		})
	}
}

func TestContractsAmendment(t *testing.T) {
	for _, method := range []string{"GET", "PUT"} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method || r.URL.Path != "/accounts/default/documents/instrument/contract" {
				t.Errorf("request = %s %s", r.Method, r.URL)
			}
			writeJSONResponse(w, `{"status":200,"data":{"resource":"document_contract_amendment","document_id":"instrument","parent_document_id":"root","relation_type":"renewal","status":"pending","number":null,"terms":{"end_date":"2028-01-01"}}}`)
		})
		r := NewContractResource(h, "default")
		var got *models.ContractResult
		var err error
		if method == "GET" {
			got, err = r.Get(context.Background(), "", "instrument")
		} else {
			relation := models.ContractRelationRenewal
			got, err = r.Set(context.Background(), "", "instrument", &models.SetDocumentContractRequest{ParentDocumentID: models.NullableValue("root"), RelationType: &relation, EndDate: models.NullableValue("2028-01-01")})
		}
		if err != nil {
			t.Fatal(err)
		}
		if got.Contract != nil || got.Amendment == nil || got.Amendment.ParentDocumentID != "root" || got.Amendment.Number != nil {
			t.Fatalf("result = %+v", got)
		}
	}
}

func TestContractsErrors(t *testing.T) {
	ctx := context.Background()
	for _, response := range []struct {
		status int
		body   string
	}{
		{400, `{"status":400,"message":"Invalid value","errors":[{"field":"value","code":"TooSmall","message":"Invalid value"}]}`},
		{403, `{"status":403,"message":"Contract tracking is not available for your current plan."}`},
		{404, `{"status":404,"message":"No contract terms."}`},
		{409, `{"status":409,"code":"ContractChanged","message":"The contract changed."}`},
	} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(response.status)
			writeJSONResponse(w, response.body)
		})
		r := NewContractResource(h, "default")
		for name, call := range map[string]func() error{
			"list": func() error { _, err := r.List(ctx, "", nil); return err }, "get": func() error { _, err := r.Get(ctx, "", "doc"); return err }, "set": func() error { _, err := r.Set(ctx, "", "doc", &models.SetDocumentContractRequest{}); return err }, "related": func() error { _, err := r.Related(ctx, "", "doc"); return err }, "renew": func() error { _, err := r.Renew(ctx, "", "doc", &models.RenewContractRequest{}); return err }, "terminate": func() error { _, err := r.Terminate(ctx, "", "doc", nil); return err },
		} {
			err := call()
			var apiErr *sdkerrors.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != response.status {
				t.Errorf("%s error = %v", name, err)
			}
		}
		if _, err := r.Get(ctx, "", " "); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("empty document = %v", err)
		}
		if _, err := NewContractResource(h, "").List(ctx, "", nil); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("empty account = %v", err)
		}
	}
}
