package resources

import (
	"context"
	"net/http"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestSignerSessionOperations(t *testing.T) {
	ctx := context.Background()
	for _, failure := range []bool{false, true} {
		for _, operation := range []string{"theme", "logo", "copy", "download", "pades", "resend", "current", "sign"} {
			t.Run(operation+map[bool]string{true: "/failure", false: "/success"}[failure], func(t *testing.T) {
				h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					assertNoClientAuth(t, r)
					method, path := http.MethodGet, ""
					switch operation {
					case "theme":
						path = "/sign/theme"
					case "logo":
						path = "/sign/logo"
					case "copy":
						method, path = http.MethodPost, "/sign/certified-copy"
					case "download":
						path = "/signers/downloads/token%2F1"
					case "pades":
						path = "/signers/downloads/token%2F1/pades"
					case "resend":
						method, path = http.MethodPost, "/signers/downloads/token%2F1/resend"
					case "current":
						path = "/signers/s%2F1/document"
					case "sign":
						path = "/sign"
					}
					if r.Method != method || r.URL.EscapedPath() != path {
						t.Errorf("request = %s %s", r.Method, r.URL)
					}
					usesCode := operation != "download" && operation != "pades" && operation != "resend"
					if (r.URL.Query().Get("signer-access-code") == "code") != usesCode {
						t.Error("wrong signer authentication")
					}
					if operation == "copy" {
						checkJSONContract(t, w, r, map[string]string{"document_id": "d2"})
					}
					if operation == "current" || operation == "sign" {
						if r.URL.Query().Get("expand") != "envelope" {
							t.Error("missing envelope expansion")
						}
						if operation == "sign" && (r.URL.Query().Get("document_id") != "d2" || r.URL.Query().Get("has_accepted_terms") != "false") {
							t.Error("missing signing options")
						}
					}
					if failure {
						w.WriteHeader(http.StatusGone)
						writeJSONResponse(w, `{"status":410,"message":"closed"}`)
						return
					}
					switch operation {
					case "theme":
						writeJSONResponse(w, `{"status":200,"data":{"name":"Workspace","primary_color":"112233","secondary_color":"ffffff","logo":null}}`)
					case "logo", "download", "pades":
						_, _ = w.Write([]byte("artifact"))
					case "copy", "resend":
						writeJSONResponse(w, `{"status":200,"data":{"available":true,"notification_method":"Email","resends_left":null,"sent_to":"s***@example.com"}}`)
					default:
						writeJSONResponse(w, `{"status":200,"data":{"id":"d2","is_signer_access_closed":true,"envelope":{"id":"e1","name":"Documents"},"certified_copy":{"available":true,"notification_method":"Email","resends_left":null}}}`)
					}
				})
				var err error
				r := NewSignerDocumentResource(h)
				switch operation {
				case "theme":
					v, e := NewSignerResource(h, "").GetTheme(ctx, "code")
					err = e
					if !failure && e == nil && v.PrimaryColor != "112233" {
						t.Error("theme lost")
					}
				case "logo", "download", "pades":
					var b []byte
					if operation == "logo" {
						b, err = NewSignerResource(h, "").DownloadLogo(ctx, "code")
					} else {
						b, err = r.DownloadByToken(ctx, "token/1", operation == "pades")
					}
					if !failure && string(b) != "artifact" {
						t.Error("artifact lost")
					}
				case "copy", "resend":
					var result *models.CertifiedCopyResult
					if operation == "copy" {
						result, err = r.RequestCertifiedCopy(ctx, "code", &models.CertifiedCopyRequest{DocumentID: "d2"})
					} else {
						result, err = r.ResendCertifiedCopy(ctx, "token/1")
					}
					if !failure && err == nil && (!result.Available || result.SentTo != "s***@example.com" || result.ResendsLeft != nil) {
						t.Error("copy response lost")
					}
				case "current", "sign":
					var result *models.Document
					accepted := false
					params := &models.SigningDocumentParams{Expand: "envelope", DocumentID: "d2", HasAcceptedTerms: &accepted}
					if operation == "current" {
						result, err = r.GetCurrentWithParams(ctx, "s/1", "code", params)
					} else {
						result, err = NewAssignmentResource(h).GetSigningInfoWithParams(ctx, "code", params)
					}
					if !failure && err == nil && ((result.IsSignerAccessClosed == nil || !*result.IsSignerAccessClosed) || result.Envelope.ID != "e1" || !result.CertifiedCopy.Available) {
						t.Error("signing context lost")
					}
				}
				if failure && !sdkerrors.IsStatusCode(err, http.StatusGone) {
					t.Fatalf("error = %v", err)
				}
				if !failure && err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestAssignmentSpendingCeilingAndEmbeddedSigner(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkJSONContract(t, w, r, map[string]any{"method": "virtual", "max_total_credits": 0, "signers": []any{map[string]any{"id": "s", "embedded": true}}})
		writeJSONResponse(w, `{"status":200,"data":{"id":"a","signing_urls":[{"signer_id":"s","url":null,"embedded":true}]}}`)
	})
	zero := 0.0
	result, err := NewAssignmentResource(h).Create(context.Background(), "d", &models.CreateAssignmentRequest{Method: models.MethodVirtual, MaxTotalCredits: &zero, Signers: []models.SignerReference{{ID: "s", Embedded: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.SigningURLs) != 1 || result.SigningURLs[0].URL != "" || !result.SigningURLs[0].Embedded {
		t.Fatal("embedded URL response lost")
	}
}

func TestTemplateIdempotencyAndSpendingCeiling(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") != "attempt-1" {
			t.Error("idempotency key missing")
		}
		checkJSONContract(t, w, r, map[string]any{"max_total_credits": 0, "signers": []any{map[string]any{"id": "s", "role_id": "role", "embedded": true}}})
		writeJSONResponse(w, `{"status":200,"data":{"id":"d"}}`)
	})
	zero := 0.0
	_, err := NewDocumentResource(h, "account").CreateFromTemplate(context.Background(), "", "template", []models.TemplateSigner{{ID: "s", RoleID: "role", Embedded: true}}, &models.CreateDocumentFromTemplateOptions{IdempotencyKey: "attempt-1", MaxTotalCredits: &zero})
	if err != nil {
		t.Fatal(err)
	}
}
