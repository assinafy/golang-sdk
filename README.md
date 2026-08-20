# Assinafy Go SDK

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)

Go client for the [Assinafy API v1](https://api.assinafy.com.br/v1/docs). The SDK exposes every operation in the current OpenAPI contract and keeps older call shapes available where that can be done without breaking users.

## Requirements

- Go 1.26 or later
- CI builds and tests with Go 1.26.x and 1.27.x; quality checks run with Go 1.27.x

Go does not designate releases as LTS. This module's compatibility floor is the `go 1.26` directive in `go.mod`; CI also covers the newer supported release.

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

### Contract-accurate call shapes

New additive methods expose v1 request or response details that older SDK signatures could not represent. This complete example shows the main variants:

```go
package example

import (
	"context"

	assinafy "github.com/assinafy/golang-sdk"
	"github.com/assinafy/golang-sdk/models"
)

func useV1Shapes(
	ctx context.Context,
	client *assinafy.Client,
	documentID string,
	fieldID string,
	email string,
) error {
	_, err := client.Assignments.EstimateCostWithRequest(ctx, documentID,
		&models.EstimateAssignmentCostRequest{
			Method: models.MethodVirtual,
			Signers: []models.EstimateAssignmentCostSigner{
				{VerificationMethod: "Email", NotificationMethods: []string{"Email"}},
			},
		})
	if err != nil {
		return err
	}

	if _, err = client.Fields.ValidateAuthenticated(ctx, "", fieldID,
		&models.ValidateFieldRequest{Value: "12345678909"}); err != nil {
		return err
	}

	if _, err = client.PublicDocuments.GetDocument(ctx, documentID); err != nil {
		return err
	}
	return client.PublicDocuments.SendTokenByEmail(ctx, documentID, email)
}
```

The corresponding compatibility choices are:

| Current v1 call | Earlier compatible call | Difference |
| --- | --- | --- |
| `PublicDocuments.GetDocument` | `PublicDocuments.Get` (deprecated) | Returns the documented full `models.Document`. |
| `PublicDocuments.SendTokenWithRequest` or `SendTokenByEmail` | `PublicDocuments.SendToken` (deprecated) | Supports the documented optional body and envelope-only response; email-bearing requests have a narrowly scoped legacy-sandbox retry. |
| `Assignments.EstimateCostWithRequest` | `Assignments.EstimateCost` (deprecated) | Uses the dedicated estimate body instead of the larger assignment-create body. |
| `Assignments.GetSigningInfoWithTerms` | `Assignments.GetSigningInfo` | Adds the documented `has_accepted_terms` query flag when it is needed. |
| `Assignments.ResetExpirationWithRequest` | `Assignments.ResetExpiration` (deprecated) | Omits an unset `expires_at` as documented; the earlier nil pointer sends legacy JSON null. |
| `Signers.AcceptTermsOnly` | `Signers.AcceptTerms` (deprecated) | Handles the documented envelope-only response. |
| `Signers.ConfirmDataAndGet` | `Signers.ConfirmData` (deprecated) | Returns the documented signer payload. |
| `Signers.UploadSignatureWithReuse` | `Signers.UploadSignature` | Adds the documented `reuse` query flag; both send PNG bytes. |
| `SignerDocuments.SearchAll` | `SignerDocuments.Search` (deprecated) | Uses only the documented `search` query and returns a non-paginated slice. |
| `Fields.ValidateAuthenticated` and `ValidateMultipleAuthenticated` | `Fields.Validate` and `ValidateMultiple` (deprecated) | Uses API-key or bearer authentication without an undocumented signer code. |
| `Documents.DetachTagWithResult` | `Documents.DetachTag` | Returns the documented `{detached}` payload. |
| `Tags.DeleteWithResult` | `Tags.Delete` | Returns the documented `{deleted}` payload. |

Earlier signatures remain available so existing programs continue to compile. Deprecated methods should be migrated when convenient; no documented API operation was removed.

## Responses, errors, and downloads

JSON responses use Assinafy's `{status,message,data}` envelope. Resource methods unwrap `data` into model values. List methods that receive pagination headers return `models.PaginatedResult[T]`; non-paginated list methods return slices.

API failures are returned as `*errors.APIError` with `StatusCode`, `Message`, and optional `Data`. Transport failures are wrapped in `*errors.NetworkError`. Use `errors.IsStatusCode(err, code)` and `errors.IsRetryable(err)` for status and retry handling.

PDF and image endpoints return raw `[]byte`. The caller is responsible for storing the bytes with the appropriate filename and permissions. Supported document artifacts are `original`, `certificated`, `certificate-page`, `pades`, and `bundle`.

## Webhooks

`NewWebhookVerifier(secret).ExtractEvent(body)` decodes a delivered body into `models.WebhookPayload`. Subscription and dispatch-history operations are available through `client.Webhooks`.

`WebhookVerifier.Verify` is a compatibility extension implementing `hex(HMAC-SHA256(secret, body))`. The current Assinafy OpenAPI contract does not define a signature header, shared secret, or HMAC algorithm, so do not treat that helper as official verification guidance without a separate agreement with Assinafy. See [compatibility extensions](docs/API.md#compatibility-extensions).

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

The current sandbox deployment returns route-level 404 for account statistics, user statistics, and user notification preferences even though the production OpenAPI declares them. Set `ASSINAFY_ALLOW_MISSING_DOCUMENTED_ROUTES=1` to skip those sandbox 404 responses; without that explicit allowance they fail. The operations remain unit-tested, documented, and available through the SDK.

| Variable | Purpose |
| --- | --- |
| `ASSINAFY_RUN_INTEGRATION_TESTS=1` | Explicitly enables the mutating integration suite. |
| `ASSINAFY_API_KEY` | Sandbox API key; required to run integration tests. |
| `ASSINAFY_ACCOUNT_ID` | Sandbox account ID; required to run integration tests. |
| `ASSINAFY_BASE_URL` | Optional override; omitted values use `assinafy.SandboxBaseURL`. |
| `ASSINAFY_ALLOW_MISSING_DOCUMENTED_ROUTES=1` | Allows the current sandbox's documented stats/preferences route 404s to skip; the manual GitHub workflow sets this explicitly. |
| `ASSINAFY_RUN_ASSIGNMENT_TESTS=1` | Creates and removes a disposable account for the full assignment flow, which sends real initial signature-request, public-token, and resend emails. |
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
