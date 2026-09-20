package assinafy_test

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	assinafy "github.com/assinafy/golang-sdk"
	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/models"
	"github.com/assinafy/golang-sdk/oauth"
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

// Example_oAuthAuthorization starts a connection: the application creates a PKCE
// pair and a state value, keeps both in the user's session, and sends the
// browser to the authorization server.
func Example_oAuthAuthorization() {
	config := &oauth.Config{
		ClientID:    "your-client-id",
		RedirectURI: "https://myapp.example/oauth/callback",
		Scopes: []string{
			oauth.ScopeDocumentsRead,
			oauth.ScopeDocumentsWrite,
			oauth.ScopeOfflineAccess,
		},
	}

	// In a real handler these come from NewPKCE and NewState and are stored in
	// the user's session; fixed values keep this example's output stable.
	authorizeURL, err := config.AuthorizationURL(oauth.AuthorizationRequest{
		State:         "random-state",
		CodeChallenge: oauth.ChallengeFor("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	parsed, _ := url.Parse(authorizeURL)
	fmt.Println(parsed.Scheme + "://" + parsed.Host + parsed.Path)
	fmt.Println(parsed.Query().Get("code_challenge_method"))
	fmt.Println(parsed.Query().Get("scope"))
	fmt.Println(parsed.Query().Get("resource"))
	// Output:
	// https://auth.assinafy.com.br/oauth/authorize
	// S256
	// documents:read documents:write offline_access
	// https://api.assinafy.com.br
}

// Example_oAuthCallback finishes a connection: the callback is validated, the
// code is exchanged for tokens, and the client is built on an auto-refreshing
// token source.
func Example_oAuthCallback() {
	config := &oauth.Config{
		ClientID:     "your-client-id",
		ClientSecret: os.Getenv("ASSINAFY_CLIENT_SECRET"),
		RedirectURI:  "https://myapp.example/oauth/callback",
	}

	callback, _ := url.Parse("https://myapp.example/oauth/callback" +
		"?code=one-time-code&state=random-state&iss=https://auth.assinafy.com.br")

	// Checking state and iss before anything else is what makes the response
	// provably yours.
	code, err := config.ParseCallback(callback, "random-state")
	fmt.Println(code, err == nil)

	// In a real handler: token, err := config.Exchange(ctx, code, verifier).
	token := &oauth.Token{AccessToken: "access-token", RefreshToken: "refresh-token"}
	source := oauth.NewTokenSource(config, token, func(renewed *oauth.Token) error {
		// Persist renewed.RefreshToken here, before the new access token is used.
		return nil
	})

	client, err := assinafy.NewClient(assinafy.ClientOptions{TokenSource: source})
	fmt.Println(err == nil, client.Documents != nil)
	// Output:
	// one-time-code true
	// true true
}

// Example_oAuthInsufficientScope shows how a call the connection lacks
// permission for names the scope to reconnect with.
func Example_oAuthInsufficientScope() {
	err := &sdkerrors.APIError{
		StatusCode: http.StatusForbidden,
		Message:    "Forbidden",
		Headers: http.Header{"Www-Authenticate": []string{
			`Bearer error="insufficient_scope", scope="documents:write", ` +
				`resource_metadata="https://api.assinafy.com.br/.well-known/oauth-protected-resource"`,
		}},
	}

	if scope, ok := sdkerrors.InsufficientScope(err); ok {
		// Send the user through the authorization flow again with this scope
		// added; retrying the call would fail the same way.
		fmt.Println("reconnect requesting:", scope)
	}
	// Output:
	// reconnect requesting: documents:write
}

// Example_digitalCertificateSigner requests a signature backed by the signer's
// own ICP-Brasil certificate.
func Example_digitalCertificateSigner() {
	request := models.CreateAssignmentRequest{
		Method: models.MethodVirtual,
		Signers: []models.SignerReference{{
			ID: "signer-id",
			// A1 and A3 certificates both reach the API this way; which one the
			// signer holds is decided in their browser by the Web PKI extension.
			VerificationMethod: models.VerificationMethodDigitalCertificate,
			// A digital-certificate signer may be notified by either channel.
			NotificationMethods: []string{models.NotificationMethodEmail},
		}},
	}

	payload, err := json.Marshal(request)
	fmt.Println(err == nil)
	fmt.Println(string(payload))
	// Output:
	// true
	// {"method":"virtual","signers":[{"id":"signer-id","verification_method":"DigitalCertificate","notification_methods":["Email"]}]}
}
