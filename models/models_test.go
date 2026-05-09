package models

import (
	"encoding/json"
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
			if tc.input != tc.expected {
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
