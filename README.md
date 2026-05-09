# Assinafy Go SDK

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)

Go client for the [Assinafy API](https://api.assinafy.com.br/v1/docs).

The SDK covers every documented Assinafy REST resource: authentication, documents (including public document flows), signers (including signer-facing flows), assignments, field definitions, templates, and webhooks.

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

// Field definitions
fields, err := client.Fields.List(ctx, "", &models.ListFieldDefinitionsParams{IncludeStandard: true})

// Webhooks
sub, err := client.Webhooks.GetSubscription(ctx, "")
```

## Webhook Verification

```go
verifier := assinafy.NewWebhookVerifier(os.Getenv("ASSINAFY_WEBHOOK_SECRET"))

if !verifier.Verify(rawBody, signature) {
    return
}

event, err := verifier.ExtractEvent(rawBody)
if err != nil {
    return
}

fmt.Println("event:", event.Event)
```

## Errors

All API failures are surfaced as `*errors.APIError` with a `StatusCode`, `Message`, and optional `Data`. Transport failures are wrapped in `*errors.NetworkError`. Helpers `errors.IsStatusCode(err, code)` and `errors.IsRetryable(err)` make it easy to react to common cases.

## Development

```bash
go test -race ./...
go vet ./...
gofmt -l .
go build ./...
```

CI runs the same checks on every push and pull request via GitHub Actions (`actions/checkout@v6`, `actions/setup-go@v6`, `actions/upload-artifact@v7`, `golangci/golangci-lint-action@v9` with `golangci-lint v2.12`).

## License

MIT — see [LICENSE](LICENSE).
