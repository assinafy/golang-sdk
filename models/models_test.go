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
	for _, tc := range []struct {
		value Timestamp
		want  string
	}{
		{Timestamp("2024-01-15T12:00:00Z"), `"2024-01-15T12:00:00Z"`},
		{Timestamp("1705316400"), `1705316400`},
		{Timestamp("001"), `"001"`},
	} {
		out, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != tc.want {
			t.Errorf("got %s, want %s", out, tc.want)
		}
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

func TestRequestModelEncoding(t *testing.T) {
	t.Run("update signer government id", func(t *testing.T) {
		governmentID := "12345678901"
		got, err := json.Marshal(UpdateSignerRequest{GovernmentID: &governmentID})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != `{"government_id":"12345678901"}` {
			t.Fatalf("body = %s", got)
		}
	})

	t.Run("empty estimate signer", func(t *testing.T) {
		got, err := json.Marshal(EstimateAssignmentCostRequest{
			Method:  MethodVirtual,
			Signers: []EstimateAssignmentCostSigner{{}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != `{"method":"virtual","signers":[{}]}` {
			t.Fatalf("body = %s", got)
		}
	})

	t.Run("open estimate entry", func(t *testing.T) {
		got, err := json.Marshal(EstimateAssignmentCostRequest{
			Method:  MethodCollect,
			Entries: []any{map[string]any{"future_field": true}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != `{"method":"collect","entries":[{"future_field":true}]}` {
			t.Fatalf("body = %s", got)
		}
	})

	t.Run("public token email", func(t *testing.T) {
		got, err := json.Marshal(SendDocumentTokenRequest{Email: "person@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != `{"email":"person@example.com"}` {
			t.Fatalf("body = %s", got)
		}
	})
}

func TestResponseModelDecoding(t *testing.T) {
	t.Run("pades artifact", func(t *testing.T) {
		var artifacts DocumentArtifacts
		if err := json.Unmarshal([]byte(`{"pades":"https://example.test/pades"}`), &artifacts); err != nil {
			t.Fatal(err)
		}
		if artifacts.PAdES == nil || *artifacts.PAdES != "https://example.test/pades" {
			t.Fatalf("PAdES = %v", artifacts.PAdES)
		}
	})

	t.Run("display settings", func(t *testing.T) {
		settings := DisplaySettings{
			Left: 0, Top: 12.5, Width: 100, Height: 20,
			FontFamily: "Arial", FontSize: 14, BackgroundColor: "#D5EBFF",
		}
		got, err := json.Marshal(settings)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(got, &decoded); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{
			"left": 0.0, "top": 12.5, "width": 100.0, "height": 20.0,
			"fontFamily": "Arial", "fontSize": 14.0, "backgroundColor": "#D5EBFF",
		}
		if !reflect.DeepEqual(decoded, want) {
			t.Fatalf("settings = %#v, want %#v", decoded, want)
		}
	})
}

func TestCreateAccountRequestMarshal(t *testing.T) {
	sender := NotificationSenderAccount
	got, err := json.Marshal(CreateAccountRequest{Name: "Acme", NotificationSenderType: &sender})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != `{"name":"Acme","notification_sender_type":"Account"}` {
		t.Errorf("body = %s", got)
	}
}

func TestDocumentStatsRowUnmarshal(t *testing.T) {
	var row DocumentStatsRow
	if err := json.Unmarshal([]byte(`{"period":"2026-06","documents_uploaded":2,"documents_sent":1,"signature_requests":3,"signature_requests_notification_email":2,"signature_requests_notification_whatsapp":1,"signature_requests_notification_bypass":1,"signature_requests_verification_email":1,"signature_requests_verification_whatsapp":1,"signature_requests_verification_bypass":0,"signature_requests_verification_digital_certificate":1,"signature_requests_viewed":2,"signature_requests_completed":1,"documents_certified":1}`), &row); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if row.Period != "2026-06" || row.DocumentsUploaded != 2 ||
		row.SignatureRequestsNotificationEmail != 2 || row.SignatureRequestsNotificationWhatsApp != 1 ||
		row.SignatureRequestsNotificationBypass != 1 || row.SignatureRequestsVerificationEmail != 1 ||
		row.SignatureRequestsVerificationWhatsApp != 1 || row.SignatureRequestsVerificationBypass != 0 ||
		row.SignatureRequestsVerificationDigitalCertificate != 1 || row.DocumentsCertified != 1 {
		t.Errorf("row = %+v", row)
	}
}

func TestNotificationPreferencesJSON(t *testing.T) {
	prefs := NotificationPreferences{
		NotificationPreferenceDocumentCompleted: true,
		NotificationPreferenceSignerDeclined:    false,
	}
	body, err := json.Marshal(prefs)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var roundTrip NotificationPreferences
	if err := json.Unmarshal(body, &roundTrip); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(roundTrip) != 2 || !roundTrip[NotificationPreferenceDocumentCompleted] || roundTrip[NotificationPreferenceSignerDeclined] {
		t.Errorf("preferences = %v", roundTrip)
	}
}
