package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
)

func TestCreditCeilingErrorPreservesAmounts(t *testing.T) {
	for _, status := range []int{http.StatusConflict, http.StatusOK} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"status":409,"code":"CreditCeilingExceeded","message":"Credit ceiling exceeded","total_credits":1.45,"max_total_credits":0}`)
		}))
		t.Cleanup(server.Close)
		client := NewHTTPClient(server.URL, "fixture-key", "", time.Second)
		_, err := client.NewRequest(http.MethodPost, "/documents/document/assignments").WithBody(map[string]any{"max_total_credits": 0}).Execute(context.Background(), nil)
		var apiErr *sdkerrors.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict || apiErr.Code != "CreditCeilingExceeded" || apiErr.TotalCredits == nil || *apiErr.TotalCredits != 1.45 || apiErr.MaxTotalCredits == nil || *apiErr.MaxTotalCredits != 0 {
			t.Fatalf("error = %#v", apiErr)
		}
		body, err := json.Marshal(apiErr)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["total_credits"] != 1.45 || decoded["max_total_credits"] != float64(0) {
			t.Fatalf("encoded error = %s", body)
		}
	}
}
