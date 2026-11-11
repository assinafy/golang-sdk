# Assinafy Go SDK

*[Leia em português](README.md) · English*

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/assinafy/golang-sdk/v2.svg)](https://pkg.go.dev/github.com/assinafy/golang-sdk/v2)

Zero-dependency Go client for all 147 published operations in the
[Assinafy API v1](https://api.assinafy.com.br/v1/docs). It requires Go 1.26+ and
TLS 1.2+ and uses only the standard library. See [docs/API.md](docs/API.md) for
every method, request field, response, authentication mode, and declared error.

## Install and configure

```bash
go get github.com/assinafy/golang-sdk/v2
```

`NewClient` accepts `APIKey`, `Token`, or `TokenSource` in that precedence order.
`AccountID` is the default workspace and `Timeout` defaults to 30 seconds.
`BaseURL` targets production; choose `SandboxBaseURL` for tests. External URLs must
use HTTPS; plain HTTP is accepted only for loopback development fixtures.

```go
client, err := assinafy.NewClient(assinafy.ClientOptions{
	APIKey: os.Getenv("ASSINAFY_API_KEY"), AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
	BaseURL: assinafy.SandboxBaseURL,
})
```

The client is safe for concurrent use. Methods accept `context.Context`, unwrap
`{status,message,data}`, return `*errors.APIError` for unsuccessful responses,
`*errors.NetworkError` for transport failures, and errors wrapping
`errors.ErrInvalidInput` for local rejection.

## Complete program: PDF to certified document

Set `ASSINAFY_API_KEY`, `ASSINAFY_ACCOUNT_ID`, `ASSINAFY_PDF_PATH`,
`ASSINAFY_SIGNER_NAME`, and `ASSINAFY_SIGNER_EMAIL`, then run `go run main.go`.
The program uses the sandbox, sends a real notification, and waits up to 30 minutes
for signing. Use an `example.com` address only when delivery is not wanted.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	assinafy "github.com/assinafy/golang-sdk/v2"
	"github.com/assinafy/golang-sdk/v2/models"
)

func must[T any](value T, err error) T {
	if err != nil { log.Fatal(err) }
	return value
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	client := must(assinafy.NewClient(assinafy.ClientOptions{
		APIKey: os.Getenv("ASSINAFY_API_KEY"), AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
		BaseURL: assinafy.SandboxBaseURL,
	}))
	pdf := must(os.ReadFile(os.Getenv("ASSINAFY_PDF_PATH")))
	document := must(client.Documents.Upload(ctx, "", pdf, "document.pdf", nil))

	email := os.Getenv("ASSINAFY_SIGNER_EMAIL")
	signer := must(client.Signers.Create(ctx, "", &models.CreateSignerRequest{
		FullName: os.Getenv("ASSINAFY_SIGNER_NAME"), Email: &email,
	}))
	estimate := must(client.Assignments.EstimateCostWithRequest(ctx, document.ID,
		&models.EstimateAssignmentCostRequest{
			Method: models.MethodVirtual,
			Signers: []models.EstimateAssignmentCostSigner{{
				VerificationMethod: models.VerificationMethodEmail,
			}},
		}))
	if !estimate.HasSufficientResources { log.Fatal(estimate.Message) }

	assignment := must(client.Assignments.Create(ctx, document.ID,
		&models.CreateAssignmentRequest{
			Method: models.MethodVirtual, MaxTotalCredits: &estimate.TotalCredits,
			Signers: []models.SignerReference{{
				ID: signer.ID, VerificationMethod: models.VerificationMethodEmail,
				NotificationMethods: []string{models.NotificationMethodEmail},
			}},
		}))
	fmt.Printf("document=%s signer=%s assignment=%s\n", document.ID, signer.ID, assignment.ID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		document = must(client.Documents.Get(ctx, document.ID))
		switch document.Status {
		case models.StatusCertificated:
			pdf = must(client.Documents.Download(ctx, document.ID, "certificated"))
			if err := os.WriteFile("document-signed.pdf", pdf, 0o600); err != nil { log.Fatal(err) }
			return
		case models.StatusExpired, models.StatusRejectedBySigner,
			models.StatusRejectedByUser, models.StatusFailed:
			log.Fatalf("signing ended with status %q", document.Status)
		}
		select {
		case <-ctx.Done(): log.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
}
```

Persist each ID as soon as it is created: upload, signer creation, and invitation
are not transactional and must not be repeated after an uncertain network result.
Reuse an existing signer because email is unique within a workspace. `collect`
assignments require pages and fields using 150-DPI coordinates; `virtual` does not.

## Verification, notification, and signing

- `Email`: email notification and OTP.
- `Whatsapp`: WhatsApp notification and OTP; the sandbox may return 403 for the plan.
- `IdDocumentFaceMatchBasic`: hosted biometric capture, a valid CPF, explicit consent,
  and workspace review when the result enters `InReview`.
- `DigitalCertificate`: ICP-Brasil A1/A3 through Web PKI, matching CPF, a virtual
  assignment, and the signer alone in its step; costs 0.5 credit plus notification.

`Assignment.SigningURLs` lets email signers request their invitation; WhatsApp-only
and embedded signers may have null URLs (empty SDK strings). A signer experience uses
`GetSigningInfo`, `GetSelf`, `VerifyOTP`, `ConfirmDataAndGet`, `AcceptTermsOnly`,
`UploadSignatureWithReuse`, and `Assignments.Sign`, always with the
`signer-access-code`. Biometrics use `GetIdentityVerification` and
`StartIdentityVerification`. Certificates use `StartCertificateSignature` and
`CompleteCertificateSignature`; their payloads are not in the published OpenAPI
description. The SDK never mixes signer codes with workspace credentials.

## Envelopes and embedded signing

`client.Envelopes` groups up to 20 documents in one invitation, with cost estimates,
`MaxTotalCredits`, and `IdempotencyKey`. Upload, wait for `metadata_ready`, estimate,
create, and follow each document until `certificated`; download the envelope ZIP.
`client.Embed` registers HTTPS origins and creates five-minute iframe sessions for
signers with `Embedded: true` and Email or WhatsApp verification. OAuth cannot mint
these sessions. Load the returned URL without logging it; each opening needs a new session.
[Payloads and flows](docs/API.md#envelope-request-and-response-payloads).

Template creation with `IdempotencyKey` delegates validation and retry recovery to the API.

## Marketplace OAuth

Use `oauth.Config` with authorization code, PKCE S256, and fresh `state` per attempt.
Validate `state` and `iss`, exchange the code immediately, save the token before use,
and connect `oauth.TokenSource` to `Client`. Refresh tokens rotate: save the new token
before exposing it, serialize refreshes per connection, and hold a storage lock across
instances. `oauth.ErrRefreshIndeterminate` requires reconnection unless storage has a
different token saved since. To disconnect, stop calls, wait for refresh/save, revoke
the latest saved refresh token, then delete it. Validate RS256 `id_token` signatures
with the issuer JWKS; the SDK does not do that.

Request only required scopes. `offline_access` enables refresh; documents, templates,
account, webhooks, OpenID, profile, and email have separate constants. Discovery uses
`DiscoverProtectedResource` and `DiscoverAuthorizationServer`; they bind `resource`
and `issuer` without redirects. External endpoints must use HTTPS.

## Webhooks and safe operation

Create an endpoint with `SigningEnabled`, persist the one-time `whsec_` secret, and
check the raw body with `NewWebhookVerifier(secret).VerifyRequest(headers, body)`.
Deduplicate by `webhook-id`, return `2xx` quickly, and process asynchronously. Rotation
invalidates the old secret; `KeepPrevious: true` allows a 24-hour overlap.
The API retries transient failures and retains seven days
of delivery history; see states, attempts, and payloads in
[docs/API.md#webhook-payloads](docs/API.md#webhook-payloads).

Do not automatically retry mutations, uploads, or notifications. Reads may use
backoff when `errors.IsRetryable` is true. Treat signing URLs, signer access codes,
tokens, and keys as credentials and never log them. Downloads return `[]byte`; write
them with restrictive permissions.

## Reference and development

- [docs/API.md](docs/API.md): 147 operations, full payloads, pagination, and binaries.
- [Go Reference](https://pkg.go.dev/github.com/assinafy/golang-sdk/v2): every exported symbol.
- [SECURITY.md](SECURITY.md): private vulnerability reporting.
- Models, contracts, fields, templates, activities, MFA, and preferences are in the
  complete reference; this README keeps the main executable flow.

```bash
go build ./... && go vet ./... && go test -race ./...
```

License: [MIT](LICENSE).
