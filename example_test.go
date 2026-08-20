package assinafy_test

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/assinafy/golang-sdk"
	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
)

func ExampleNewClient() {
	client, err := assinafy.NewClient(assinafy.ClientOptions{
		APIKey:    os.Getenv("ASSINAFY_API_KEY"),
		AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
	})

	fmt.Println(err == nil)
	fmt.Println(client.Documents != nil && client.Signers != nil)
	// Output:
	// true
	// true
}

func ExampleNewClient_sandbox() {
	client, err := assinafy.NewClient(assinafy.ClientOptions{
		APIKey:    os.Getenv("ASSINAFY_API_KEY"),
		AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
		BaseURL:   assinafy.SandboxBaseURL,
		Timeout:   15 * time.Second,
	})

	fmt.Println(err == nil)
	fmt.Println(client.PublicDocuments != nil)
	// Output:
	// true
	// true
}

func Example_requestJSON() {
	message := "Please review and sign"
	request := models.CreateAssignmentRequest{
		Method: models.MethodCollect,
		Signers: []models.SignerReference{{
			ID:                  "signer-id",
			VerificationMethod:  "Email",
			NotificationMethods: []string{"Email"},
		}},
		Entries: []models.AssignmentEntry{{
			PageID: "page-id",
			Fields: []models.AssignmentField{{
				SignerID: "signer-id",
				FieldID:  "field-id",
				DisplaySettings: models.DisplaySettings{
					Left: 20, Top: 40, Width: 180, Height: 30, FontSize: 12,
				},
			}},
		}},
		Message: &message,
	}

	payload, err := json.Marshal(request)
	fmt.Println(err == nil)
	fmt.Println(string(payload))
	// Output:
	// true
	// {"method":"collect","signers":[{"id":"signer-id","verification_method":"Email","notification_methods":["Email"]}],"entries":[{"page_id":"page-id","fields":[{"signer_id":"signer-id","field_id":"field-id","display_settings":{"left":20,"top":40,"width":180,"height":30,"fontSize":12}}]}],"message":"Please review and sign"}
}

func Example_pagination() {
	params := models.ListParams{PerPage: 250}
	params.SetDefaults()
	page := models.PaginatedResult[models.Document]{
		Data: []models.Document{{ID: "document-id"}},
		Pagination: models.PaginationMeta{
			CurrentPage: 1,
			TotalCount:  42,
			PageCount:   1,
			PerPage:     params.PerPage,
		},
	}

	fmt.Printf("request page=%d per-page=%d\n", params.Page, params.PerPage)
	fmt.Printf("received=%d total=%d\n", len(page.Data), page.Pagination.TotalCount)
	// Output:
	// request page=1 per-page=100
	// received=1 total=42
}

func Example_errors() {
	_, err := assinafy.NewClient(assinafy.ClientOptions{BaseURL: "://invalid"})
	fmt.Println(stderrors.Is(err, assinafy.ErrInvalidBaseURL))

	apiErr := &sdkerrors.APIError{
		StatusCode: http.StatusTooManyRequests,
		Message:    "rate limit exceeded",
	}
	fmt.Println(sdkerrors.IsStatusCode(apiErr, http.StatusTooManyRequests))
	fmt.Println(sdkerrors.IsRetryable(apiErr))
	// Output:
	// true
	// true
	// true
}
