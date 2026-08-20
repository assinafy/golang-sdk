package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestListParamsSetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    ListParams
		expected ListParams
	}{
		{"zero values", ListParams{}, ListParams{Page: 1, PerPage: 25}},
		{"custom page", ListParams{Page: 5}, ListParams{Page: 5, PerPage: 25}},
		{"custom per-page", ListParams{PerPage: 50}, ListParams{Page: 1, PerPage: 50}},
		{"per-page over max", ListParams{PerPage: 150}, ListParams{Page: 1, PerPage: 100}},
		{"all set", ListParams{Page: 2, PerPage: 50, Search: "test", Sort: "-created_at"}, ListParams{Page: 2, PerPage: 50, Search: "test", Sort: "-created_at"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.SetDefaults()
			if !reflect.DeepEqual(tc.input, tc.expected) {
				t.Errorf("got %+v, want %+v", tc.input, tc.expected)
			}
		})
	}
}

func TestDocumentStatusValues(t *testing.T) {
	pairs := map[DocumentStatus]string{
		StatusUploading:          "uploading",
		StatusUploaded:           "uploaded",
		StatusMetadataProcessing: "metadata_processing",
		StatusMetadataReady:      "metadata_ready",
		StatusExpired:            "expired",
		StatusCertificating:      "certificating",
		StatusCertificated:       "certificated",
		StatusRejectedBySigner:   "rejected_by_signer",
		StatusPendingSignature:   "pending_signature",
		StatusPendingSignatures:  "pending_signatures",
		StatusPartiallySigned:    "partially_signed",
		StatusReady:              "ready",
		StatusRejectedByUser:     "rejected_by_user",
		StatusFailed:             "failed",
	}
	for k, v := range pairs {
		if string(k) != v {
			t.Errorf("status %q != %q", k, v)
		}
	}
}

func TestAssignmentMethodValues(t *testing.T) {
	if string(MethodVirtual) != "virtual" {
		t.Errorf("MethodVirtual = %q", MethodVirtual)
	}
	if string(MethodCollect) != "collect" {
		t.Errorf("MethodCollect = %q", MethodCollect)
	}
}

func TestTimestampUnmarshal(t *testing.T) {
	cases := map[string]string{
		`"2024-01-15T12:00:00Z"`: "2024-01-15T12:00:00Z",
		`1705316400`:             "1705316400",
		`null`:                   "",
	}
	for input, want := range cases {
		var ts Timestamp
		if err := json.Unmarshal([]byte(input), &ts); err != nil {
			t.Fatalf("%s: unexpected error: %v", input, err)
		}
		if ts.String() != want {
			t.Errorf("%s: got %q, want %q", input, ts, want)
		}
	}

	for _, input := range []string{`true`, `[]`, `{}`} {
		var ts Timestamp
		if err := json.Unmarshal([]byte(input), &ts); err == nil {
			t.Errorf("%s: expected invalid timestamp error", input)
		}
	}
}

func TestTimestampMarshal(t *testing.T) {
	ts := Timestamp("2024-01-15T12:00:00Z")
	out, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `"2024-01-15T12:00:00Z"` {
		t.Errorf("got %s, want quoted string", out)
	}
}

func TestTimestampIsZero(t *testing.T) {
	if !Timestamp("").IsZero() || Timestamp("2024-01-15T12:00:00Z").IsZero() {
		t.Fatal("IsZero must report only the empty timestamp as zero")
	}
}

func TestPayloadUnmarshal(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  Payload
	}{
		{"empty array", `[]`, nil},
		{"null", `null`, nil},
		{"object", `{"foo":"bar"}`, Payload{"foo": "bar"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p Payload
			if err := json.Unmarshal([]byte(tc.input), &p); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(p) != len(tc.want) {
				t.Fatalf("got %v, want %v", p, tc.want)
			}
			for k, v := range tc.want {
				if p[k] != v {
					t.Errorf("key %q = %v, want %v", k, p[k], v)
				}
			}
		})
	}

	t.Run("non-empty array rejected", func(t *testing.T) {
		var p Payload
		if err := json.Unmarshal([]byte(`["x"]`), &p); err == nil {
			t.Fatal("expected an error for non-empty array")
		}
	})
}

func TestPayloadMarshal(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Payload
		want string
	}{
		{"nil", nil, `null`},
		{"empty object", Payload{}, `{}`},
		{"object", Payload{"foo": "bar"}, `{"foo":"bar"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil || string(got) != tc.want {
				t.Fatalf("Marshal() = %s, %v; want %s", got, err, tc.want)
			}
		})
	}
}

func TestCostEstimateDecodesBlockingReasonAndMessage(t *testing.T) {
	// Matches the live estimate-cost envelope data shape.
	const body = `{"documents":1,"credits":0,"total_credits":0,"breakdown":[],` +
		`"document_balance":73,"credit_balance":0,"has_sufficient_resources":true,` +
		`"blocking_reason":null,"message":null}`
	var ce CostEstimate
	if err := json.Unmarshal([]byte(body), &ce); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ce.Documents != 1 || !ce.HasSufficientResources {
		t.Errorf("unexpected decode: %+v", ce)
	}
	if ce.BlockingReason != nil || ce.Message != nil {
		t.Errorf("expected nil blocking_reason/message, got %v / %v", ce.BlockingReason, ce.Message)
	}

	const blocked = `{"documents":1,"blocking_reason":"InsufficientCredits",` +
		`"message":"Account does not have enough credits."}`
	var ce2 CostEstimate
	if err := json.Unmarshal([]byte(blocked), &ce2); err != nil {
		t.Fatalf("unmarshal blocked: %v", err)
	}
	if ce2.BlockingReason == nil || *ce2.BlockingReason != "InsufficientCredits" {
		t.Errorf("blocking_reason = %v", ce2.BlockingReason)
	}
	if ce2.Message == nil || *ce2.Message == "" {
		t.Errorf("message = %v", ce2.Message)
	}
}

func TestTemplateSignerStepMarshal(t *testing.T) {
	step := 2
	out, err := json.Marshal(TemplateSigner{RoleID: "r1", ID: "s1", Step: &step})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(out); got != `{"role_id":"r1","id":"s1","step":2}` {
		t.Errorf("got %s", got)
	}
	// Step omitted when nil.
	out, _ = json.Marshal(TemplateSigner{RoleID: "r1"})
	if got := string(out); got != `{"role_id":"r1"}` {
		t.Errorf("nil step should be omitted, got %s", got)
	}
}

func TestUpdateTagRequestMarshal(t *testing.T) {
	name := "Renamed"
	color := "112233"
	cases := []struct {
		name string
		req  UpdateTagRequest
		want string
	}{
		{"name only", UpdateTagRequest{Name: &name}, `{"name":"Renamed"}`},
		{"color only", UpdateTagRequest{Color: &color}, `{"color":"112233"}`},
		{"name and color", UpdateTagRequest{Name: &name, Color: &color}, `{"color":"112233","name":"Renamed"}`},
		{"clear color", UpdateTagRequest{ClearColor: true}, `{"color":null}`},
		{"clear color wins over color", UpdateTagRequest{Color: &color, ClearColor: true}, `{"color":null}`},
		{"empty leaves all unchanged", UpdateTagRequest{}, `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(out) != tc.want {
				t.Errorf("got %s, want %s", out, tc.want)
			}
		})
	}
}

func TestCreateTagRequestMarshal(t *testing.T) {
	color := "112233"
	for _, tc := range []struct {
		name string
		req  CreateTagRequest
		want string
	}{
		{"omit color", CreateTagRequest{Name: "Tag"}, `{"name":"Tag"}`},
		{"set color", CreateTagRequest{Name: "Tag", Color: &color}, `{"color":"112233","name":"Tag"}`},
		{"null color", CreateTagRequest{Name: "Tag", ClearColor: true}, `{"color":null,"name":"Tag"}`},
		{"null wins", CreateTagRequest{Name: "Tag", Color: &color, ClearColor: true}, `{"color":null,"name":"Tag"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.req)
			if err != nil || string(got) != tc.want {
				t.Fatalf("Marshal() = %s, %v; want %s", got, err, tc.want)
			}
		})
	}
}

func TestUpdateFieldDefinitionRequestMarshalRegex(t *testing.T) {
	regex := "^[0-9]+$"
	for _, tc := range []struct {
		name string
		req  UpdateFieldDefinitionRequest
		want string
	}{
		{"omit", UpdateFieldDefinitionRequest{}, `{}`},
		{"set", UpdateFieldDefinitionRequest{Regex: &regex}, `{"regex":"^[0-9]+$"}`},
		{"clear", UpdateFieldDefinitionRequest{ClearRegex: true}, `{"regex":null}`},
		{"clear wins", UpdateFieldDefinitionRequest{Regex: &regex, ClearRegex: true}, `{"regex":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCreateFieldDefinitionRequestMarshalRegex(t *testing.T) {
	regex := "^[0-9]+$"
	for _, tc := range []struct {
		name string
		req  CreateFieldDefinitionRequest
		want string
	}{
		{"omit", CreateFieldDefinitionRequest{Type: "text", Name: "Code"}, `{"type":"text","name":"Code"}`},
		{"set", CreateFieldDefinitionRequest{Type: "text", Name: "Code", Regex: &regex}, `{"type":"text","name":"Code","regex":"^[0-9]+$"}`},
		{"clear", CreateFieldDefinitionRequest{Type: "text", Name: "Code", ClearRegex: true}, `{"type":"text","name":"Code","regex":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestWebhookDispatchListParamsSetDefaults(t *testing.T) {
	p := WebhookDispatchListParams{}
	p.SetDefaults()
	if p.Page != 1 || p.PerPage != 20 {
		t.Errorf("expected defaults Page=1 PerPage=20, got %+v", p)
	}
	p = WebhookDispatchListParams{PerPage: 500}
	p.SetDefaults()
	if p.PerPage != 100 {
		t.Errorf("expected PerPage clamped to 100, got %d", p.PerPage)
	}
}
