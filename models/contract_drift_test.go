package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestContractRequestModels(t *testing.T) {
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

func TestContractResponseModels(t *testing.T) {
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
