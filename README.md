# Assinafy Go SDK

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)

Go client for the [Assinafy API](https://api.assinafy.com.br/v1/docs).

The SDK covers every documented Assinafy REST resource: authentication, documents (including public document flows and document tags), signers (including signer-facing flows), assignments, field definitions, templates, tags, and webhooks.

## Requirements

- Go 1.26 or later

## Installation

```bash
go get github.com/assinafy/golang-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    assinafy "github.com/assinafy/golang-sdk"
    "github.com/assinafy/golang-sdk/models"
)

func main() {
    client, err := assinafy.NewClient(assinafy.ClientOptions{
        APIKey:    os.Getenv("ASSINAFY_API_KEY"),
        AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    pdf, err := os.ReadFile("contract.pdf")
    if err != nil {
        log.Fatal(err)
    }

    result, err := client.UploadAndRequestSignatures(
        ctx, pdf, "contract.pdf",
        []models.UploadAndRequestSignaturesSigner{
            {Name: "John Doe", Email: "john@example.com"},
        },
        "Please sign this contract",
        nil, "",
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("document:", result.Document.ID)
    fmt.Println("assignment:", result.Assignment.ID)
}
```

## Configuration

| Option          | Type            | Default                              | Description                                                                  |
| --------------- | --------------- | ------------------------------------ | ---------------------------------------------------------------------------- |
| `APIKey`        | `string`        | empty                                | Permanent credential. Sent as `X-Api-Key`.                                   |
| `Token`         | `string`        | empty                                | Access token. Sent as `Authorization: Bearer`. Used when `APIKey` is empty.  |
| `AccountID`     | `string`        | empty                                | Default workspace ID for account-scoped resources.                           |
| `BaseURL`       | `string`        | `https://api.assinafy.com.br/v1`     | API base URL. Use `assinafy.SandboxBaseURL` for the sandbox environment.    |
| `Timeout`       | `time.Duration` | `30s`                                | HTTP client timeout.                                                         |

Credentials are optional at construction time so unauthenticated endpoints (login, public document lookup, signer-access-code flows) can be used.

## Resources

```go
ctx := context.Background()

// Authentication
auth, err := client.Authentication.Login(ctx, &models.LoginRequest{
    Email: "user@example.com", Password: "secret",
})

// Documents
doc, err := client.Documents.Upload(ctx, "", pdfBytes, "contract.pdf", nil)
docs, err := client.Documents.List(ctx, "", &models.ListParams{Search: "contract"})
// Filter by tag IDs (AND semantics — only documents carrying every listed tag):
tagged, err := client.Documents.List(ctx, "", &models.ListParams{Tags: []string{tagID1, tagID2}})
verified, err := client.Documents.Verify(ctx, "SIGNATURE_HASH")

// Signers
email := "john@example.com"
signer, err := client.Signers.Create(ctx, "", &models.CreateSignerRequest{
    FullName: "John Doe", Email: &email,
})

// Assignments
assignment, err := client.Assignments.Create(ctx, doc.ID, &models.CreateAssignmentRequest{
    Method: models.MethodVirtual,
    Signers: []models.SignerReference{
        {ID: signer.ID, VerificationMethod: "Email", NotificationMethods: []string{"Email"}},
    },
})

// Templates
templates, err := client.Templates.List(ctx, "", &models.ListParams{Search: "Service"})

// Tags (workspace labels) and document tags
tag, err := client.Tags.Create(ctx, "", &models.CreateTagRequest{Name: "Contracts"})
attached, err := client.Documents.ReplaceTags(ctx, "", doc.ID, []string{"Contracts", "2026-Q1"})

// Field definitions
fields, err := client.Fields.List(ctx, "", &models.ListFieldDefinitionsParams{IncludeStandard: true})

// Webhooks
sub, err := client.Webhooks.GetSubscription(ctx, "")
```

## Webhook Verification

`ExtractEvent` decodes a delivered webhook body into a typed `models.WebhookPayload`:

```go
verifier := assinafy.NewWebhookVerifier(os.Getenv("ASSINAFY_WEBHOOK_SECRET"))

event, err := verifier.ExtractEvent(rawBody)
if err != nil {
    return
}

fmt.Println("event:", event.Event)
```

> **`Verify` is experimental.** Assinafy's documented [delivery contract](https://api.assinafy.com.br/v1/docs) specifies only the HTTP method, content type, and retry/circuit-breaker behaviour — it does not define a signature header, and the subscription object exposes no shared-secret field. `Verify` implements the conventional `hex(HMAC-SHA256(secret, body))` scheme, but confirm the exact header and algorithm with Assinafy before relying on it for authenticity.

## Errors

All API failures are surfaced as `*errors.APIError` with a `StatusCode`, `Message`, and optional `Data`. Transport failures are wrapped in `*errors.NetworkError`. Helpers `errors.IsStatusCode(err, code)` and `errors.IsRetryable(err)` make it easy to react to common cases.

## API Coverage

Every endpoint documented at <https://api.assinafy.com.br/v1/docs> is exposed by the SDK:

| Area | Endpoints | SDK |
| --- | --- | --- |
| Authentication | `POST /login`, `POST /authentication/social-login`, `POST/GET/DELETE /users/api-keys`, `PUT /authentication/{change,request,reset}-password` | `client.Authentication` |
| Signers (workspace) | `POST/GET /accounts/{id}/signers`, `GET/PUT/DELETE /accounts/{id}/signers/{sid}` | `client.Signers` |
| Signers (self-service) | `GET /signers/self`, `PUT /signers/accept-terms`, `POST /verify`, `PUT /documents/{id}/signers/confirm-data`, `POST/GET /signature[/{type}]` | `client.Signers` |
| Documents | `POST/GET /accounts/{id}/documents`, `GET/DELETE /documents/{id}`, `GET /documents/{id}/{thumbnail,download/{art},pages/{pid}/download}`, `GET /documents/{hash}/verify`, `GET /documents/{id}/activities`, `GET /documents/statuses`, `POST /accounts/{id}/templates/{tid}/documents[/estimate-cost]` | `client.Documents` |
| Document tags | `GET/PUT/POST /accounts/{id}/documents/{did}/tags`, `DELETE /accounts/{id}/documents/{did}/tags/{tid}` | `client.Documents.{ListTags,ReplaceTags,AppendTags,DetachTag}` |
| Public documents | `GET /public/documents/{id}`, `PUT /public/documents/{id}/send-token` | `client.PublicDocuments` |
| Templates | `GET /accounts/{id}/templates`, `GET /accounts/{id}/templates/{tid}` | `client.Templates` |
| Tags | `GET/POST /accounts/{id}/tags`, `PUT/DELETE /accounts/{id}/tags/{tid}` | `client.Tags` |
| Assignments | `POST /documents/{id}/assignments[/estimate-cost]`, `PUT /documents/{id}/assignments/{aid}/{reset-expiration,reject}`, `POST /documents/{id}/assignments/{aid}`, `PUT /documents/{id}/assignments/{aid}/signers/{sid}/resend`, `POST .../estimate-resend-cost`, `GET .../whatsapp-notifications`, `GET /sign` | `client.Assignments` |
| Signer documents | `GET /signers/{sid}/document[s]`, `PUT /signers/documents/{sign,decline}-multiple`, `GET /signers/{sid}/documents/{id}/download/{art}` | `client.SignerDocuments` |
| Field definitions | `POST/GET /accounts/{id}/fields`, `GET/PUT/DELETE /accounts/{id}/fields/{fid}`, `POST .../validate[-multiple]`, `GET /field-types` | `client.Fields` |
| Webhooks | `GET/PUT /accounts/{id}/webhooks/subscriptions`, `PUT /accounts/{id}/webhooks/inactivate`, `GET /accounts/{id}/webhooks`, `POST /accounts/{id}/webhooks/{did}/retry`, `GET /webhooks/event-types` | `client.Webhooks` |

Subscriptions are turned off with `client.Webhooks.Inactivate`. The `DELETE /accounts/{id}/webhooks/subscriptions` route mentioned in passing by the docs is not implemented by the live API (it returns 404), so the SDK does not expose it.

## Development

```bash
go test -race ./...
go vet ./...
gofmt -l .
go build ./...
```

CI runs the same checks on every push and pull request via GitHub Actions (`actions/checkout@v6`, `actions/setup-go@v6`, `actions/upload-artifact@v7`, `golangci/golangci-lint-action@v9` with `golangci-lint v2.12`).

### Integration tests

Tests prefixed `TestIntegration` hit the live API and are skipped unless both `ASSINAFY_API_KEY` and `ASSINAFY_ACCOUNT_ID` are set. They cover the read-only endpoints, full create/get/update/delete lifecycles for signers, tags, and field definitions, a document upload + estimate-cost round trip, artifact/page downloads, signature-hash verification, the document-tag attach/detach flow, and the webhook subscription update/inactivate lifecycle.

| Variable | Purpose |
| --- | --- |
| `ASSINAFY_API_KEY` | API key (required to run). |
| `ASSINAFY_ACCOUNT_ID` | Workspace/account ID (required to run). |
| `ASSINAFY_BASE_URL` | Optional base URL override; set to `https://sandbox.assinafy.com.br/v1` to target the sandbox. Defaults to production. |
| `ASSINAFY_RUN_ASSIGNMENT_TESTS` | Set to `1` to also run the full virtual-assignment lifecycle. **This sends real signature-request emails**, so it is opt-in. |

```bash
ASSINAFY_API_KEY=... ASSINAFY_ACCOUNT_ID=... \
ASSINAFY_BASE_URL=https://sandbox.assinafy.com.br/v1 \
  go test -race -run '^TestIntegration' -v .
```

## License

MIT — see [LICENSE](LICENSE).
