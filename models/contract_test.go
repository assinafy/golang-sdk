package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestContractRequestFieldStates(t *testing.T) {
	for _, tc := range []struct {
		name string
		body any
		want string
	}{
		{"omitted", SetDocumentContractRequest{}, `{}`},
		{"null", SetDocumentContractRequest{
			Value: &Nullable[json.Number]{}, EndDate: &Nullable[string]{},
			AutoRenew: &Nullable[bool]{}, Metadata: &Nullable[map[string]string]{},
			ParentDocumentID: &Nullable[string]{},
		}, `{"value":null,"end_date":null,"auto_renew":null,"metadata":null,"parent_document_id":null}`},
		{"zero", SetDocumentContractRequest{
			Value: NullableValue(json.Number("0.00")), AutoRenew: NullableValue(false),
			NoticePeriodDays: NullableValue(0), ExternalReferences: NullableValue([]ContractExternalReference{}),
		}, `{"value":0.00,"auto_renew":false,"notice_period_days":0,"external_references":[]}`},
		{"precision", RenewContractRequest{Value: NullableValue(json.Number("9999999999999.99"))}, `{"value":9999999999999.99}`},
		{"renew clear", RenewContractRequest{Value: &Nullable[json.Number]{}}, `{"value":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatal(err)
			}
			var actual, expected map[string]json.RawMessage
			if err := json.Unmarshal(got, &actual); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.want), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("body = %s, want %s", got, tc.want)
			}
		})
	}
	if _, err := json.Marshal(NullableValue(json.Number("invalid"))); err == nil {
		t.Fatal("invalid monetary value accepted")
	}
}

func TestContractResultVariants(t *testing.T) {
	var result ContractResult
	for _, tc := range []struct {
		body      string
		amendment bool
	}{
		{`{"resource":"document_contract","document_id":"doc","status":"active","value":9999999999999.99,"auto_renew":false,"term":1,"external_references":[],"metadata":null,"amendments":[]}`, false},
		{`{"resource":"document_contract_amendment","document_id":"instrument","parent_document_id":"doc","relation_type":"amendment","status":"pending","number":null,"terms":{"value":123.45},"changes":{"value":{"from":null,"to":123.45}}}`, true},
	} {
		if err := json.Unmarshal([]byte(tc.body), &result); err != nil {
			t.Fatal(err)
		}
		if tc.amendment {
			if result.Contract != nil || result.Amendment == nil || result.Amendment.Number != nil || string(result.Amendment.Changes["value"].To) != "123.45" {
				t.Fatalf("amendment = %+v", result)
			}
		} else if result.Amendment != nil || result.Contract == nil || result.Contract.Value == nil || result.Contract.Value.String() != "9999999999999.99" {
			t.Fatalf("contract = %+v", result)
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		var before, after map[string]json.RawMessage
		if err := json.Unmarshal([]byte(tc.body), &before); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &after); err != nil {
			t.Fatal(err)
		}
		for key, want := range before {
			if string(after[key]) != string(want) {
				t.Errorf("%s = %s, want %s", key, after[key], want)
			}
		}
	}
	if err := json.Unmarshal([]byte(`null`), &result); err != nil || result.Contract != nil || result.Amendment != nil {
		t.Fatalf("null = %+v, %v", result, err)
	}
	for _, body := range []string{`[]`, `{"term":"bad"}`, `{"resource":"document_contract_amendment","number":"bad"}`} {
		if err := json.Unmarshal([]byte(body), &result); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
