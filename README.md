# Assinafy Go SDK

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)

Go client for the [Assinafy API v1](https://api.assinafy.com.br/v1/docs), covering all 89 operations in the published OpenAPI contract.

## Requirements

- Go 1.26 or later
- CI builds and tests with Go 1.26.x and 1.27.x; quality checks run with Go 1.27.x

Go does not designate releases as LTS. This module's minimum version is the `go 1.26` directive in `go.mod`; CI also covers the newer supported release.

## Installation

```bash
go get github.com/assinafy/golang-sdk
```

## Quick start

This complete program makes a read-only sandbox request. Keep credentials in environment variables; do not place them in source code.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	assinafy "github.com/assinafy/golang-sdk"
)

func main() {
	client, err := assinafy.NewClient(assinafy.ClientOptions{
		APIKey:    os.Getenv("ASSINAFY_API_KEY"),
		AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
		BaseURL:   assinafy.SandboxBaseURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	accounts, err := client.Accounts.List(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("accounts: %d\n", len(accounts))
}
```

## Configuration

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `APIKey` | `string` | empty | Permanent credential, sent as `X-Api-Key`. |
| `Token` | `string` | empty | Access token, sent as `Authorization: Bearer`; used when `APIKey` is empty. |
| `AccountID` | `string` | empty | Default account ID for account-scoped resources. A non-empty method argument overrides it. |
| `BaseURL` | `string` | `https://api.assinafy.com.br/v1` | API base URL. Set `assinafy.SandboxBaseURL` for sandbox use. |
| `Timeout` | `time.Duration` | `30s` | HTTP client timeout. |

Credentials are optional when constructing a client because login, public-document, verification, and signer-access-code operations do not use account credentials. When both account credentials are configured, `APIKey` takes precedence over `Token`.

## API coverage

The [complete API mapping](docs/API.md) documents all 89 operations from the current OpenAPI description, including method and path, authentication, SDK call, request content and fields, response payload, pagination or binary behavior, errors, and a direct official link for each operation.

| Official area | Operations | SDK resource |
| --- | ---: | --- |
| Accounts | 10 | `client.Accounts` |
| Assignments | 7 | `client.Assignments` |
| Authentication | 9 | `client.Authentication` |
| Documents | 18 | `client.Documents` |
| Fields | 8 | `client.Fields` |
| Signers | 5 | `client.Signers` |
| Signing | 17 | `client.PublicDocuments`, `client.Signers`, `client.Assignments`, `client.SignerDocuments` |
| Tags | 4 | `client.Tags` |
| Templates | 1 | `client.Templates` |
| Users | 4 | `client.Users` |
| Webhooks | 6 | `client.Webhooks` |
| **Total** | **89** | |

Common workflows can also use `client.UploadAndRequestSignatures`, which uploads a PDF, creates each signer, and creates the assignment in one call.

## Document signing flow

A standard virtual-signature request has five stages: upload the PDF, wait for its page metadata, create the signer, estimate and create the assignment, then download the certified artifact after every signer finishes. Uploads must be PDFs no larger than 25 MB and 2,000 pages. The SDK rejects empty, oversized, or non-PDF input before sending it; the API enforces the page limit.

The following functions show the owner-side flow. Keep recipient data outside source control and pass it into the application at runtime.

```go
package signing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	assinafy "github.com/assinafy/golang-sdk"
	"github.com/assinafy/golang-sdk/models"
)

func createRequest(
	ctx context.Context,
	client *assinafy.Client,
	pdf []byte,
	filename, signerName, signerEmail string,
) (*models.Document, *models.Assignment, error) {
	document, err := client.Documents.Upload(ctx, "", pdf, filename, nil)
	if err != nil {
		return nil, nil, err
	}

	document, err = waitForMetadata(ctx, client, document.ID)
	if err != nil {
		return document, nil, err
	}

	signer, err := client.Signers.Create(ctx, "", &models.CreateSignerRequest{
		FullName: signerName,
		Email:    &signerEmail,
	})
	if err != nil {
		return document, nil, err
	}

	estimate, err := client.Assignments.EstimateCostWithRequest(ctx, document.ID,
		&models.EstimateAssignmentCostRequest{
			Method: models.MethodVirtual,
			Signers: []models.EstimateAssignmentCostSigner{{
				VerificationMethod:  "Email",
				NotificationMethods: []string{"Email"},
			}},
		})
	if err != nil {
		return document, nil, err
	}
	if !estimate.HasSufficientResources {
		return document, nil, errors.New("assignment cannot be funded")
	}

	assignment, err := client.Assignments.Create(ctx, document.ID, &models.CreateAssignmentRequest{
		Method: models.MethodVirtual,
		Signers: []models.SignerReference{{
			ID:                  signer.ID,
			VerificationMethod:  "Email",
			NotificationMethods: []string{"Email"},
		}},
	})
	if err != nil {
		return document, nil, err
	}
	return document, assignment, nil
}

func waitForMetadata(ctx context.Context, client *assinafy.Client, documentID string) (*models.Document, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		document, err := client.Documents.Get(ctx, documentID)
		if err != nil {
			return nil, err
		}
		switch document.Status {
		case models.StatusMetadataReady:
			return document, nil
		case models.StatusFailed:
			return document, errors.New("document processing failed")
		}
		select {
		case <-ctx.Done():
			return document, ctx.Err()
		case <-ticker.C:
		}
	}
}

func downloadWhenCertified(ctx context.Context, client *assinafy.Client, documentID, destination string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		document, err := client.Documents.Get(ctx, documentID)
		if err != nil {
			return err
		}
		switch document.Status {
		case models.StatusCertificated:
			pdf, err := client.Documents.Download(ctx, documentID, "certificated")
			if err != nil {
				return err
			}
			return os.WriteFile(destination, pdf, 0o600)
		case models.StatusExpired, models.StatusRejectedBySigner, models.StatusRejectedByUser, models.StatusFailed:
			return fmt.Errorf("signing ended with status %q", document.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
```

Creating the assignment may send its notification immediately. The returned `Assignment.SigningURLs` contains signer-specific URLs when the API includes them; treat those URLs and every `signer-access-code` as credentials and never log them. A signer normally completes the browser flow from that URL. For an application-owned signer session, use the methods on `client.Signers`, `client.Assignments`, and `client.SignerDocuments` with the supplied signer access code.

`UploadAndRequestSignatures` provides the upload, signer-creation, and virtual-assignment stages as one sequential helper. It is not transactional: when a later API call fails, the returned partial result contains the uploaded document and created signer IDs so the caller can perform lifecycle-appropriate cleanup.

## Responses, errors, and downloads

JSON responses use Assinafy's `{status,message,data}` envelope. Resource methods unwrap `data` into model values. List methods that receive pagination headers return `models.PaginatedResult[T]`; non-paginated list methods return slices.

API failures are returned as `*errors.APIError` with `StatusCode`, `Message`, optional `Data`, response headers, and account-deletion `Restrictions`. Transport failures are wrapped in `*errors.NetworkError`. Invalid required paths and file payloads wrap `errors.ErrInvalidInput` before a network request. Cross-origin redirects for mutations wrap the non-retryable `errors.ErrUnsafeRedirect`; read-only redirects strip API and signer credentials. Use `errors.IsStatusCode(err, code)` and `errors.IsRetryable(err)` for status and retry handling.

PDF and image endpoints return raw `[]byte`. The caller is responsible for storing the bytes with the appropriate filename and permissions. Supported document artifacts are `original`, `certificated`, `certificate-page`, `pades`, and `bundle`.

## Webhooks

`NewWebhookVerifier(secret).ExtractEvent(body)` decodes a delivered body into `models.WebhookPayload`. Subscription and dispatch-history operations are available through `client.Webhooks`.

`WebhookVerifier.Verify` implements `hex(HMAC-SHA256(secret, body))`. Use it only when Assinafy has separately confirmed that signature scheme and its header for your integration.

## Development

```bash
go build ./...
go test -race ./...
go vet ./...
gofmt -l .
```

GitHub Actions run these checks for pushes and pull requests using pinned action revisions. The repository may be mirrored from GitLab; GitHub remains the CI execution target represented by the badge above.

### Integration tests

`TestIntegration*` tests are skipped unless `ASSINAFY_RUN_INTEGRATION_TESTS=1` and both `ASSINAFY_API_KEY` and `ASSINAFY_ACCOUNT_ID` are set. They default to `https://sandbox.assinafy.com.br/v1`, and they refuse the production host unless `ASSINAFY_RUN_PRODUCTION_TESTS=1` is also set.

The standard integration suite is not read-only: it creates, updates, and deletes temporary sandbox resources and, when the route is available, round-trips a notification preference with cleanup. Use a disposable sandbox account. Two higher-impact flows require separate opt-in flags.

| Variable | Purpose |
| --- | --- |
| `ASSINAFY_RUN_INTEGRATION_TESTS=1` | Explicitly enables the mutating integration suite. |
| `ASSINAFY_API_KEY` | Sandbox API key; required to run integration tests. |
| `ASSINAFY_ACCOUNT_ID` | Sandbox account ID; required to run integration tests. |
| `ASSINAFY_BASE_URL` | Optional override; omitted values use `assinafy.SandboxBaseURL`. |
| `ASSINAFY_RUN_ASSIGNMENT_TESTS=1` | Creates and removes a disposable account for the full assignment flow, which sends real initial signature-request, public-token, and resend emails. |
| `ASSINAFY_TEST_EMAIL_PRIMARY` | First runtime-only recipient; required with assignment tests. |
| `ASSINAFY_TEST_EMAIL_SECONDARY` | Second runtime-only recipient; required with assignment tests. |
| `ASSINAFY_RUN_ACCOUNT_ADMIN_TESTS=1` | Enables account create/update/logo/delete lifecycle coverage. |
| `ASSINAFY_RUN_PRODUCTION_TESTS=1` | Removes the production safety guard. It does not make mutating tests read-only. |

```bash
ASSINAFY_RUN_INTEGRATION_TESTS=1 ASSINAFY_API_KEY=... ASSINAFY_ACCOUNT_ID=... \
  go test -race -run '^TestIntegration' -v .
```

Never enable production execution or the opt-in flows unless their side effects are intended. Signer-facing methods that require a live `signer-access-code` are contract-tested locally and exercised live only within an appropriate signing session.

The suite does not create or revoke the credential it is currently using, change/reset a user's password, or fabricate social-provider, password-reset, or signer-access tokens. Those operations have strict local method/path/auth/body/response contract tests; positive live tests require the corresponding purpose-specific credentials.

## License

MIT — see [LICENSE](LICENSE).
