# Assinafy Go SDK

Go client SDK for the [Assinafy API](https://api.assinafy.com.br/v1/docs) — a Brazilian digital signature platform.

Covers documents, signers, assignments, webhooks, workspaces, and templates.

## Requirements

- Go 1.26+

## Installation

```bash
go get github.com/assinafy/assinafy-go
```

## Quick Start

```go
package main

import (
    "context"
    assinafy "github.com/assinafy/assinafy-go"
)

func main() {
    client, err := assinafy.NewClient(assinafy.ClientOptions{
        APIKey:    "your-api-key",
        AccountID: "your-account-id",
    })
    if err != nil {
        panic(err)
    }

    result, err := client.UploadAndRequestSignatures(
        context.Background(),
        fileContent,
        "contract.pdf",
        []assinafy.UploadAndRequestSignaturesSigner{
            {Name: "John Doe", Email: "john@example.com"},
        },
        "Please sign this contract",
        nil,
        "",
    )
    if err != nil {
        panic(err)
    }

    println("Document ID:", result.Document.ID)
}
```

## Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `APIKey` | string | — | Preferred credential (sent as `X-Api-Key`). |
| `Token` | string | — | Legacy access token (sent as `Bearer`). |
| `AccountID` | string | — | Default workspace/account ID. |
| `BaseURL` | string | `https://api.assinafy.com.br/v1` | API base URL. Use sandbox URL for testing. |
| `WebhookSecret` | string | — | Shared secret used by `WebhookVerifier`. |
| `Timeout` | time.Duration | `30s` | Request timeout. |

## Resources

### Documents

```go
// Upload from bytes
doc, err := client.Documents.Upload(ctx, accountID, fileContent, "contract.pdf", nil)

// List documents
page, err := client.Documents.List(ctx, accountID, &ListParams{Page: 1, PerPage: 20})

// Get document details
doc, err := client.Documents.Get(ctx, documentID)

// Download artifact
content, err := client.Documents.Download(ctx, documentID, "certificated")

// Delete document
err = client.Documents.Delete(ctx, documentID)

// Create from template
doc, err := client.Documents.CreateFromTemplate(ctx, accountID, templateID, signers, opts)
```

### Signers

```go
// Create signer
signer, err := client.Signers.Create(ctx, accountID, &CreateSignerRequest{
    FullName: "John Doe",
    Email:    "john@example.com",
})

// List signers
page, err := client.Signers.List(ctx, accountID, &ListParams{Search: "john"})

// Get signer
signer, err := client.Signers.Get(ctx, accountID, signerID)

// Update signer
signer, err := client.Signers.Update(ctx, accountID, signerID, &UpdateSignerRequest{
    FullName: "John Smith",
})

// Delete signer
err = client.Signers.Delete(ctx, accountID, signerID)
```

### Assignments

```go
// Create assignment
assignment, err := client.Assignments.Create(ctx, documentID, &CreateAssignmentRequest{
    Method:    MethodVirtual,
    SignerIDs: []string{signerID},
    Message:   "Please sign",
})

// Estimate cost
cost, err := client.Assignments.EstimateCost(ctx, documentID, req)

// Resend notification
err = client.Assignments.ResendNotification(ctx, documentID, assignmentID, signerID)

// Cancel
err = client.Assignments.Cancel(ctx, documentID, assignmentID, "reason")

// Decline
err = client.Assignments.Decline(ctx, documentID, assignmentID, "reason")
```

### Webhooks

```go
// Register webhook
sub, err := client.Webhooks.Register(ctx, accountID, &RegisterWebhookRequest{
    URL:    "https://example.com/webhook",
    Email:  "admin@example.com",
    Events: []string{"document_ready", "signer_signed_document"},
})

// Get current subscription
sub, err := client.Webhooks.Get(ctx, accountID)

// List event types
types, err := client.Webhooks.ListEventTypes(ctx)

// List dispatches
page, err := client.Webhooks.ListDispatches(ctx, accountID, params)

// Retry dispatch
err = client.Webhooks.RetryDispatch(ctx, accountID, dispatchID)

// Inactivate webhook
err = client.Webhooks.Inactivate(ctx, accountID)
```

### Webhook Verification

```go
verifier := assinafy.NewWebhookVerifier(webhookSecret)

// Verify signature
if !verifier.Verify(rawBody, signature) {
    return ErrUnauthorized
}

// Extract event
event, _ := verifier.ExtractEvent(rawBody)
eventType := verifier.GetEventType(event)
data := verifier.GetEventData(event)
```

### Workspaces

```go
// Create workspace
ws, err := client.Workspaces.Create(ctx, &CreateWorkspaceRequest{Name: "My Workspace"})

// List workspaces
list, err := client.Workspaces.List(ctx)

// Get workspace
ws, err := client.Workspaces.Get(ctx, accountID)

// Update workspace
ws, err = client.Workspaces.Update(ctx, accountID, &UpdateWorkspaceRequest{Name: "New Name"})

// Delete workspace
err = client.Workspaces.Delete(ctx, accountID)
```

### Templates

```go
// List templates
page, err := client.Templates.List(ctx, accountID, &ListParams{Search: "NDA"})

// Get template details
template, err := client.Templates.Get(ctx, accountID, templateID)
```

## High-Level Helper

Upload a PDF, create or reuse signers by email, and start a virtual signature assignment:

```go
result, err := client.UploadAndRequestSignatures(
    ctx,
    fileContent,
    "contract.pdf",
    []assinafy.UploadAndRequestSignaturesSigner{
        {Name: "John", Email: "john@example.com"},
        {Name: "Jane", Email: "jane@example.com", WhatsAppPhoneNumber: "+5548999990000"},
    },
    "Please sign",
    nil,
    "",
)
```

## Error Handling

The SDK returns typed errors:

```go
import "github.com/assinafy/assinafy-go/errors"

// Check error type
var apiErr *errors.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
}

// Check retryable
if errors.IsRetryable(err) {
    // retry
}

// Check specific status code
if errors.IsStatusCode(err, 404) {
    // not found
}
```

## Development

```bash
# Run tests
go test ./...

# Build
go build ./...
```

## License

MIT
