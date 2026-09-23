# Assinafy Go SDK

*[Leia em português](README.md) · English*

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/assinafy/golang-sdk.svg)](https://pkg.go.dev/github.com/assinafy/golang-sdk)

Go client for the [Assinafy API v1](https://api.assinafy.com.br/v1/docs). It covers
all 93 operations in the published OpenAPI description with typed requests,
typed responses, and a single error contract.

## Contents

1. [Requirements](#requirements)
2. [Installation](#installation)
3. [Quick start](#quick-start)
4. [Configuring the client](#configuring-the-client)
5. [How every call behaves](#how-every-call-behaves)
6. [The signature lifecycle](#the-signature-lifecycle)
   - [1. Upload the document](#1-upload-the-document)
   - [2. Wait for page metadata](#2-wait-for-page-metadata)
   - [3. Create the signers](#3-create-the-signers)
   - [4. Estimate the cost](#4-estimate-the-cost)
   - [5. Request the signatures](#5-request-the-signatures)
   - [6. Track the request](#6-track-the-request)
   - [7. Drive a signer session](#7-drive-a-signer-session)
   - [8. Download the certified document](#8-download-the-certified-document)
   - [The whole flow in one call](#the-whole-flow-in-one-call)
7. [Verifying and notifying signers](#verifying-and-notifying-signers)
   - [Signing with an ICP-Brasil certificate](#signing-with-an-icp-brasil-certificate)
8. [Connecting other people's workspaces with OAuth](#connecting-other-peoples-workspaces-with-oauth)
9. [Creating documents from templates](#creating-documents-from-templates)
10. [Organizing documents with tags and fields](#organizing-documents-with-tags-and-fields)
11. [Receiving webhooks](#receiving-webhooks)
12. [Accounts, users, and statistics](#accounts-users-and-statistics)
13. [Errors and retries](#errors-and-retries)
14. [Handling credentials safely](#handling-credentials-safely)
15. [API coverage](#api-coverage)
16. [Development](#development)
17. [License](#license)

## Requirements

- Go 1.26 or later. The module's minimum is the `go 1.26` directive in `go.mod`.
- No third-party dependencies; the SDK uses only the standard library.

Go does not designate releases as LTS: the Go team supports the two most recent
major releases. CI builds and tests on both (Go 1.26.x and 1.27.x) and runs the
formatting, vet, vulnerability, and lint checks on Go 1.27.x.

## Installation

```bash
go get github.com/assinafy/golang-sdk
```

## Quick start

This complete program makes one read-only sandbox request. Keep credentials in
environment variables; never place them in source code.

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

## Configuring the client

`assinafy.NewClient` validates its options and builds a concurrency-safe client
without making a network request. One client can be shared across goroutines for
the lifetime of a process.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `APIKey` | `string` | empty | Permanent credential, sent as `X-Api-Key`. |
| `Token` | `string` | empty | Access token, sent as `Authorization: Bearer`; used when `APIKey` is empty. |
| `TokenSource` | `assinafy.TokenSource` | nil | Resolves a bearer token per request, used when `APIKey` and `Token` are both empty. This is how an OAuth connection authenticates; see [Connecting other people's workspaces with OAuth](#connecting-other-peoples-workspaces-with-oauth). |
| `AccountID` | `string` | empty | Default account ID for account-scoped resources. A non-empty method argument overrides it. |
| `BaseURL` | `string` | `https://api.assinafy.com.br/v1` | API root. Use `assinafy.SandboxBaseURL` for the sandbox. |
| `Timeout` | `time.Duration` | `30s` | Per-request HTTP timeout. |

**Choosing a credential.** An API key is the right choice for automating **your
own** workspace: it is permanent and scoped to the user who created it. A bearer
token comes from `client.Authentication.Login` and expires. A `TokenSource` is
for an application acting in **someone else's** workspace with their permission.
When more than one is configured, `APIKey` wins, then `Token`, then
`TokenSource`. Credentials are optional at construction time, because login,
public-document, verification, and signer-access-code operations do not use one.

**Account selection.** Account-scoped methods take an `accountID` argument.
Passing `""` falls back to `ClientOptions.AccountID`; if both are empty the SDK
rejects the call locally rather than sending a malformed path.

**Resources.** The client groups the API by area:

```go
client.Accounts        client.Documents   client.Templates  client.Tags
client.Signers         client.Assignments client.Fields     client.Webhooks
client.SignerDocuments client.Users       client.Authentication
client.PublicDocuments
```

The OAuth flow lives in its own `oauth` package, because an integration calls it
before it has a client.

## How every call behaves

The same handful of rules apply to every method, so they are worth reading once.

- **Context first.** Every method takes a `context.Context` and honours
  cancellation and deadlines.
- **Envelope unwrapping.** The API wraps JSON responses as
  `{status, message, data}`. Methods return the `data` payload decoded into a
  model; you never handle the envelope. The four OAuth operations are the
  exception: they answer with plain OAuth JSON and are never wrapped.
- **Pagination.** List endpoints that return `X-Pagination-*` headers give you a
  `models.PaginatedResult[T]` with `Data` and a `Pagination` struct
  (`CurrentPage`, `TotalCount`, `PageCount`, `PerPage`). Endpoints without those
  headers return a plain slice. `models.ListParams.SetDefaults` normalises `Page`
  to 1 and clamps `PerPage` to the API maximum of 100.
- **Binary payloads.** Artifact, page, thumbnail, logo, and signature-image
  endpoints return raw `[]byte`. Storing them with a suitable filename and file
  mode is the caller's responsibility.
- **Optional request fields.** Request models use pointers for fields the API
  treats as optional; `nil` omits the field and leaves the server value
  unchanged. Fields with an explicit three-state contract (a tag colour, a field
  regex) expose a `Clear…` boolean that sends JSON `null`.
- **Errors.** Every failure is typed. See [Errors and retries](#errors-and-retries).

## The signature lifecycle

A virtual signature request — the common case — moves a document through eight
stages. Each stage below is one SDK call, in order.

### 1. Upload the document

Uploads must be PDFs of at most 25 MB and 2,000 pages. The SDK rejects empty,
oversized, and non-PDF input before sending anything; the page limit is enforced
server-side.

```go
document, err := client.Documents.Upload(ctx, "", pdfBytes, "agreement.pdf", nil)
```

The document is created immediately and processed asynchronously.

### 2. Wait for page metadata

A `virtual` assignment may be created while the document is still processing. A
`collect` assignment places fields on specific pages, so it needs the page
metadata first. Poll until the status settles:

```go
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
```

`client.Documents.ListStatuses` returns every status code the API can report and
whether documents in it can be deleted.

### 3. Create the signers

Signers belong to the workspace, not to a document, so they can be reused. Look
one up with `client.Signers.List(ctx, accountID, &models.ListParams{Search: …})`
before creating a duplicate.

```go
signer, err := client.Signers.Create(ctx, "", &models.CreateSignerRequest{
	FullName: signerName,
	Email:    &signerEmail,
})
```

A signer needs an email address, a WhatsApp number in E.164 form, or both,
depending on how they will be verified and notified.

### 4. Estimate the cost

Each assignment consumes one document from the plan's monthly allowance; if that
is exhausted, an extra document is charged in credits. Verification and
notification methods have their own prices. Estimating first turns a later
billing failure into a decision you control:

```go
estimate, err := client.Assignments.EstimateCostWithRequest(ctx, document.ID,
	&models.EstimateAssignmentCostRequest{
		Method: models.MethodVirtual,
		Signers: []models.EstimateAssignmentCostSigner{{
			VerificationMethod:  "Email",
			NotificationMethods: []string{"Email"},
		}},
	})
if err != nil {
	return err
}
if !estimate.HasSufficientResources {
	return fmt.Errorf("cannot fund assignment: %v", estimate.BlockingReason)
}
```

`BlockingReason` is `PendingPayment`, `InsufficientDocuments`, or
`InsufficientCredits` when the operation is blocked, and `nil` otherwise.

### 5. Request the signatures

```go
assignment, err := client.Assignments.Create(ctx, document.ID, &models.CreateAssignmentRequest{
	Method: models.MethodVirtual,
	Signers: []models.SignerReference{{
		ID:                  signer.ID,
		VerificationMethod:  models.VerificationMethodEmail,
		NotificationMethods: []string{models.NotificationMethodEmail},
	}},
})
```

Both method fields may be omitted; see
[Verifying and notifying signers](#verifying-and-notifying-signers) for what each
one does and costs.

Choose the method by whether the signer fills anything in:

- `models.MethodVirtual` — the signer accepts the document as-is. No fields.
- `models.MethodCollect` — the signer completes fields you position on specific
  pages through `CreateAssignmentRequest.Entries`, each field carrying a
  `models.DisplaySettings` placement in the API's 150-DPI page coordinates.

Set `SignerReference.Step` to sign in sequence. Signers sharing a step are
notified together; the next step is notified only after the previous one
finishes. When any signer has a step, all of them need one, numbered
contiguously from 1.

Creating the assignment can dispatch notifications immediately. The response's
`SigningURLs` holds one signer-specific URL per virtual-flow signer.

### 6. Track the request

| Question | Call |
| --- | --- |
| What has happened to this document? | `client.Documents.Activities(ctx, documentID)` |
| Did the WhatsApp message go out, and what did it say? | `client.Assignments.ListWhatsAppNotifications(ctx, documentID, assignmentID)` |
| Can I nudge one signer again? | `client.Assignments.EstimateResendCost(…)` then `client.Assignments.ResendNotification(…)` |
| The deadline passed — can I extend it? | `client.Assignments.ResetExpirationWithRequest(…)` |
| What is outstanding across the account? | `client.Assignments.List(ctx, params)` |

### 7. Drive a signer session

Most integrations send the signer to their `SigningURL` and let the browser flow
handle the rest. When your own application hosts the signer experience, the same
steps are available with the signer's access code, which the SDK sends as the
`signer-access-code` query parameter:

```go
document, err := client.Assignments.GetSigningInfo(ctx, signerAccessCode)          // GET /sign
signer, err := client.Signers.GetSelf(ctx, signerAccessCode)                       // who is signing
err = client.Signers.VerifyEmail(ctx, signerAccessCode, otp)                       // one-time code
_, err = client.Signers.ConfirmDataAndGet(ctx, documentID, signerAccessCode, body) // confirm details
err = client.Signers.AcceptTermsOnly(ctx, signerAccessCode)                        // accept the terms
err = client.Signers.UploadSignatureWithReuse(ctx, signerAccessCode, "signature", true, pngBytes)
err = client.Assignments.Sign(ctx, documentID, assignmentID, signerAccessCode, items)
```

A signer with several pending documents can act on them together through
`client.SignerDocuments.SignMultiple` and `DeclineMultiple`, and can list or
search their own documents with `client.SignerDocuments.List` and `SearchAll`.
Declining a single assignment is `client.Assignments.Decline`.

For virtual assignments the signer must confirm their data before signing, so
call `ConfirmDataAndGet` ahead of `Sign`. A signer verified by digital
certificate takes a different route at the last step; see
[Signing with an ICP-Brasil certificate](#signing-with-an-icp-brasil-certificate).

### 8. Download the certified document

```go
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
		case models.StatusExpired, models.StatusRejectedBySigner,
			models.StatusRejectedByUser, models.StatusFailed:
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

Five artifacts are available:

| Artifact | Contents |
| --- | --- |
| `original` | The PDF exactly as uploaded. |
| `certificated` | The signed PDF with Assinafy's certification page. |
| `certificate-page` | The certification page on its own. |
| `pades` | PAdES-signed PDF; present only when a signer used a digital certificate. |
| `bundle` | A zip of the other available artifacts. |

`client.Documents.Thumbnail` and `DownloadPage` return rendered images, and
`client.Documents.Verify` checks a signature hash without any credential.

### The whole flow in one call

`Client.UploadAndRequestSignatures` performs stages 1, 3, and 5 in sequence:

```go
result, err := client.UploadAndRequestSignatures(ctx, pdfBytes, "agreement.pdf",
	[]models.UploadAndRequestSignaturesSigner{{Name: signerName, Email: signerEmail}},
	"Please review and sign", nil, "")
```

It validates every name, email, WhatsApp number, and expiry before the first
request. It is **not** transactional: when a later call fails, the returned
partial result still carries the uploaded `Document` and the created `SignerIDs`
so the caller can clean up.

## Verifying and notifying signers

Each signer on an assignment has a **verification method** — how they prove who
they are before signing — and a **notification method** — how they are told a
signature is wanted. The two are coupled: send one, both, or neither, and the
missing side is inferred. Send neither and both default to `Email`. Only one
notification method is allowed per signer.

| Verification | Allowed notification | Requires | Cost per signer |
| --- | --- | --- | --- |
| `models.VerificationMethodEmail` *(default)* | `Email` | An email address on the signer | 0 credits |
| `models.VerificationMethodWhatsApp` | `Whatsapp` | A `whatsapp_phone_number`, and a paid subscription | 0.45 credits |
| `models.VerificationMethodDigitalCertificate` | `Email` or `Whatsapp` | The Digital Certificate feature, and a CPF or CNPJ in the signer's `government_id` | 2 credits plus the notification |

No verification method is billed on its own — you pay for the notification it is
paired with. Choosing WhatsApp verification therefore also chooses the WhatsApp
notification, and its 0.45 credits. `DigitalCertificate` is the exception: the
signature itself costs 2 credits per signer on top of the notification, charged
when the assignment is created. `client.Assignments.EstimateCostWithRequest`
returns the exact total before you commit to it, itemized in `Breakdown`.

An invalid pairing is rejected with `400`, so use the constants rather than
string literals.

### Signing with an ICP-Brasil certificate

A `DigitalCertificate` signer signs with their own ICP-Brasil certificate — **A1**
held in software or **A3** on a token or smart card — through the Web PKI browser
extension, producing a qualified **PAdES** signature on the document itself.

Three conditions apply. The account needs the Digital Certificate feature
(Standard and Pro plans). The signer needs a CPF or CNPJ in `government_id`,
which `client.Signers.Update` sets: a CPF requires that person's own certificate
(an e-CPF, or an e-CNPJ naming them as legal representative), and a CNPJ requires
an e-CNPJ for that company. And each certificate signer must be **alone in their
signing step**.

```go
_, err = client.Signers.Update(ctx, "", signer.ID, &models.UpdateSignerRequest{
	GovernmentID: &cpf,
})

assignment, err := client.Assignments.Create(ctx, document.ID, &models.CreateAssignmentRequest{
	Method: models.MethodVirtual,
	Signers: []models.SignerReference{{
		ID:                  signer.ID,
		VerificationMethod:  models.VerificationMethodDigitalCertificate,
		NotificationMethods: []string{models.NotificationMethodEmail},
		Step:                &firstStep,
	}},
})
```

Such a signer confirms their data and accepts the terms as usual — one
`ConfirmDataAndGet` call can do both — but `client.Assignments.Sign` **rejects
them** with `400`. Their signature comes from a two-step exchange with the
browser extension instead:

```go
start, err := client.Signers.StartCertificateSignature(ctx, signerAccessCode)
// Hand start.Token to the Web PKI extension, which signs it with the
// signer's certificate and returns the signature.
result, err := client.Signers.CompleteCertificateSignature(ctx, signerAccessCode,
	&models.CompleteCertificateSignatureRequest{Token: start.Token, Signature: signature})
// result.SignerName is the name read from the certificate.
```

Once the document closes, its `pades` artifact carries the qualified signatures.

> These two routes are named in the reference but have no published request or
> response schema. The SDK implements the shapes Assinafy's own signing flow
> uses; `CompleteCertificateSignatureRequest.Extra` sends any further field a
> deployment expects.

## Connecting other people's workspaces with OAuth

Use OAuth when you build a product that **other Assinafy customers connect to
their own workspace**. The user approves your application once and you receive a
token limited to the permissions they approved and to the single workspace they
picked — you never handle their password or API key, and they can disconnect you
at any time. Automating your own workspace needs none of this; keep using an API
key.

Register the application in the Assinafy app under **Settings → OAuth
applications**. You need to be an owner of the workspace that will own it, and
its plan must include OAuth applications. Registration yields a `client_id`, plus
a `client_secret` for a **confidential** application whose code runs on a server
you control. A **public** application — one running on the user's device, which
cannot keep a secret — gets no secret and authenticates with PKCE alone.
Applications cannot be created through the API.

The `oauth` package implements the whole flow on the standard library.

**Step 1 — start a connection.** Create a fresh PKCE pair and state for every
attempt, keep both in the user's session, and send the browser to the URL with a
full page navigation:

```go
config := &oauth.Config{
	ClientID:     os.Getenv("ASSINAFY_CLIENT_ID"),
	ClientSecret: os.Getenv("ASSINAFY_CLIENT_SECRET"), // omit for a public app
	RedirectURI:  "https://myapp.example/oauth/callback",
	Scopes: []string{
		oauth.ScopeDocumentsRead,
		oauth.ScopeDocumentsWrite,
		oauth.ScopeOfflineAccess,
	},
}

pkce, err := oauth.NewPKCE()
state, err := oauth.NewState()
// Persist pkce.Verifier and state against this user's session.

authorizeURL, err := config.AuthorizationURL(oauth.AuthorizationRequest{
	State:         state,
	CodeChallenge: pkce.Challenge,
})
http.Redirect(w, r, authorizeURL, http.StatusFound)
```

**Step 2 — handle the return.** `ParseCallback` checks the `state` in constant
time and the `iss` parameter before returning anything, and reports a declined
approval as an `*oauth.Error` with code `access_denied`. The code is single-use
and expires 60 seconds after approval, so exchange it at once:

```go
code, err := config.ParseCallback(r.URL, sessionState)
if err != nil {
	// oauth.ErrorCode(err) == oauth.ErrCodeAccessDenied when the user declined
	return err
}
token, err := config.Exchange(ctx, code, sessionVerifier)
```

**Step 3 — find the workspace and call the API.** A token belongs to the one
workspace the user chose, so `Accounts.List` returns exactly that workspace.
Store its ID alongside the token:

```go
source := oauth.NewTokenSource(config, token, func(renewed *oauth.Token) error {
	return store.SaveTokens(userID, renewed) // before the new access token is used
})

client, err := assinafy.NewClient(assinafy.ClientOptions{TokenSource: source})
workspaces, err := client.Accounts.List(ctx)
client, err = assinafy.NewClient(assinafy.ClientOptions{
	TokenSource: source,
	AccountID:   workspaces[0].ID,
})
```

The token source renews an expiring access token from the refresh token before
each request. Access tokens last an hour; a connection lasts **30 days from the
user's approval**, and refreshing does not extend it, so plan for users to
reconnect monthly.

> **Every refresh rotates the refresh token and retires the old one.** Replaying
> a retired refresh token cannot be told apart from a stolen one, so the server
> ends the whole connection. Save the new token in `onRefresh` before anything
> else — returning an error there aborts the refresh, so your storage is never
> left holding a dead token — treat a timeout as "maybe it worked" and re-read
> what you saved, and refresh one connection at a time. `TokenSource` serializes
> its own refreshes, so concurrent callers share one.

**Permissions.** Request the minimum; every extra permission is another line the
user reads before deciding, and they approve everything or nothing.

| Constant | Lets your application |
| --- | --- |
| `oauth.ScopeDocumentsRead` | Read documents, their signers, assignments, and activity |
| `oauth.ScopeDocumentsWrite` | Create documents and send them for signature — this can spend the workspace's notification credits |
| `oauth.ScopeTemplatesRead` | Read templates |
| `oauth.ScopeTemplatesWrite` | Create and change templates |
| `oauth.ScopeAccountRead` | Read the workspace's profile, theme, and logo |
| `oauth.ScopeWebhooksWrite` | Configure and deactivate the workspace webhook subscription |
| `oauth.ScopeOpenID` | Receive an `id_token` identifying the user, and call `UserInfo` |
| `oauth.ScopeProfile` | Read the user's name |
| `oauth.ScopeEmail` | Read the user's email and whether it is verified |
| `oauth.ScopeOfflineAccess` | Receive a refresh token, to keep working while the user is away |

Read `Token.Scope` for what was actually granted rather than assuming. Billing,
workspace membership, credential management, and administration are never
reachable with an OAuth token, whatever its scopes. Calling an endpoint without
the scope it needs returns `403` with a challenge naming the missing one:

```go
if scope, ok := sdkerrors.InsufficientScope(err); ok {
	// Reconnect requesting scope; retrying the call would fail the same way.
}
```

A `403` without that header has another cause: a different workspace, the user's
own role, or an area OAuth cannot reach. A token works only for the workspace it
was issued for — calling any other returns `403`, even one the same user belongs
to — so a customer with several workspaces connects each one separately.

**Signing users in.** With `ScopeOpenID` the token carries an `id_token`, an
RS256 JWT saying who approved. Validate it with any OpenID Connect library
against the issuer's JWKS, then read the user's name and email from
`config.UserInfo(ctx, accessToken)`.

**Disconnecting.** When a user disconnects in your product, revoke the token
rather than only deleting it. Revocation always answers success:

```go
err := config.Revoke(ctx, refreshToken, oauth.HintRefreshToken)
```

**Discovery.** The endpoint URLs are published by the authorization server, and
`Config.Endpoint` defaults to production, so most integrations configure nothing.
To read them at runtime:

```go
resource, err := oauth.DiscoverProtectedResource(ctx, "", nil)          // this API
metadata, err := oauth.DiscoverAuthorizationServer(ctx, resource.Issuer(), nil)
config.Endpoint = metadata.Endpoint()
```

Before going live: a new PKCE pair and state per attempt; `state` and `iss`
checked; the secret only on your server; the rotated refresh token saved before
use; `401` handled by refreshing and, failing that, asking the user to reconnect;
every production redirect URI registered, `https://`, and matched exactly. New
applications are unverified — the approval screen says so and they can connect to
at most 25 workspaces — so ask Assinafy to verify yours before launching beyond a
pilot. The authorize and token endpoints accept 50 requests per minute per IP.

## Creating documents from templates

A template carries pre-placed fields and named signing roles, so generating a
document and its assignment is one call. Templates are created in the Assinafy
web application; the SDK reads them and generates from them.

```go
templates, err := client.Templates.List(ctx, "", nil)
template, err := client.Templates.Get(ctx, "", templates.Data[0].ID) // roles, pages, default tags

estimate, err := client.Documents.EstimateCostFromTemplate(ctx, "", template.ID,
	[]models.TemplateSigner{{RoleID: template.Roles[0].ID, VerificationMethod: "Email"}})

document, err := client.Documents.CreateFromTemplate(ctx, "", template.ID,
	[]models.TemplateSigner{{RoleID: template.Roles[0].ID, ID: signer.ID}},
	&models.CreateDocumentFromTemplateOptions{
		Name: "Agreement for Example Ltd",
		Tags: []string{"contracts"},
	})
```

Supply one `TemplateSigner` per role, referencing signers that already exist in
the account. `CreateDocumentFromTemplateOptions.EditorFields` fills the
template's editor values, and its `Tags` are merged with the template's default
document tags.

## Organizing documents with tags and fields

**Tags** are workspace labels. Names are unique per account, case-insensitively,
so creating a duplicate returns `409`.

```go
tag, err := client.Tags.Create(ctx, "", &models.CreateTagRequest{Name: "contracts"})
_, err = client.Documents.AppendTags(ctx, "", document.ID, []string{tag.ID})
_, err = client.Documents.ReplaceTags(ctx, "", document.ID, []string{tag.ID})
err = client.Documents.DetachTag(ctx, "", document.ID, tag.ID)
```

Filter by tag with `models.ListParams{Tags: []string{tagID}}`, which the SDK
sends as a comma-separated list. Tag filters use AND semantics: a document must
carry every listed tag. `client.Tags.Delete(ctx, "", tagID, force)` removes a
tag, detaching it from its documents when `force` is true.

**Fields** are the typed inputs a `collect` assignment asks signers to complete.
`client.Fields.ListTypes` returns the available types; `client.Fields.Create`,
`Get`, `Update`, and `Delete` manage the account's definitions, and
`client.Fields.ValidateAuthenticated` and `ValidateMultipleAuthenticated` check
values against a definition before you submit them.

## Receiving webhooks

Point Assinafy at an endpoint, choose the events, and handle the deliveries.

```go
events, err := client.Webhooks.ListEventTypes(ctx)
_, err = client.Webhooks.UpdateSubscription(ctx, "", &models.UpdateWebhookSubscriptionRequest{
	Events:   []string{"document_ready", "signer_signed_document"},
	IsActive: true,
	URL:      "https://example.com/hooks/assinafy",
	Email:    "ops@example.com",
})
```

All four fields are required by the API and are always sent. `client.Webhooks.Inactivate`
is the documented way to stop deliveries without discarding the configuration.

Decode a delivered body with the webhook verifier:

```go
verifier := assinafy.NewWebhookVerifier(sharedSecret)
event, err := verifier.ExtractEvent(requestBody)
```

`client.Webhooks.ListDispatches` returns the delivery history — status code,
response body, and error per attempt — filterable by event, delivery result, and
Unix time range, and `client.Webhooks.RetryDispatch` forces another attempt.

Deduplicate on `WebhookPayload.ID`, accept unknown fields and event names, and do
not rely on ordering between events. The full event catalogue with each event's
subject, object, and payload keys is in
[docs/API.md](docs/API.md#webhook-payloads).

`WebhookVerifier.Verify` implements `hex(HMAC-SHA256(secret, body))`. The
published contract documents no signature header, so use `Verify` only after
Assinafy has confirmed that scheme and header for your integration.

## Accounts, users, and statistics

`client.Accounts` reads and administers workspaces: `List`, `Get`, `Create`,
`Update`, `Delete`, plus `GetTheme`, `DownloadLogo`, `UploadLogo`, and
`DeleteLogo` for signer-facing branding. Deleting an account is destructive and
returns its blockers in `APIError.Restrictions` unless you pass `force`.

`client.Users` covers the authenticated user: `GetSelf`, plus
`GetNotificationPreferences` and `UpdateNotificationPreferences` for the nine
owner-facing email notifications. An update merges the keys you send and returns
the complete map.

`client.Accounts.Stats` and `client.Users.Stats` return the document funnel —
uploaded, sent, requested, viewed, completed, certified — either monthly (the
last twelve months) or daily for one `YYYY-MM` month. Both series are
zero-filled.

`client.Authentication` handles `Login`, `SocialLogin`, `LinkSocialLogin`, the
password workflows, and the `CreateAPIKey`/`GetAPIKey`/`DeleteAPIKey` lifecycle.
Creating a key replaces the previous one and is the only time the full key is
returned. `SocialLoginURL` builds the browser URL that starts a provider's OAuth
flow and makes no HTTP request of its own.

## Errors and retries

Every failure is one of three kinds, and each is inspectable:

```go
document, err := client.Documents.Get(ctx, documentID)

var apiErr *sdkerrors.APIError
switch {
case sdkerrors.IsStatusCode(err, http.StatusNotFound):
	// the document does not exist
case stderrors.As(err, &apiErr):
	// any other API response: apiErr.StatusCode, .Message, .Data, .Headers, .Restrictions
case sdkerrors.IsRetryable(err):
	// 429, 5xx, or a transport failure — retry with backoff
case stderrors.Is(err, sdkerrors.ErrInvalidInput):
	// rejected locally, before any request was sent
}
```

- `*errors.APIError` — a non-success HTTP response or envelope status. Carries
  `StatusCode`, `Message`, optional `Data`, the response `Headers` (including
  `Retry-After`), and account-deletion `Restrictions`.
- `*errors.NetworkError` — a transport failure. Any `signer-access-code` in the
  URL is redacted before the error is returned.
- `errors.ErrInvalidInput` — a local rejection: an empty required ID, a
  non-PDF upload, an oversized file, a malformed expiry.

On an OAuth connection, `errors.InsufficientScope(err)` reads the scope named by
a `403` `insufficient_scope` challenge, and the `oauth` package returns
`*oauth.Error` carrying the RFC 6749 code — read it with `oauth.ErrorCode(err)`.

`errors.IsRetryable` reports `true` for HTTP 429, 5xx, and transport failures,
and `false` for cancelled contexts and refused redirects. The SDK does not retry
on your behalf: retry policy, backoff, and idempotency belong to the caller, and
several operations here send email or spend credits.

`errors.ErrUnsafeRedirect` wraps a cross-origin redirect that the SDK refused to
follow for a request carrying a body or a non-idempotent method. Read-only
cross-origin redirects are followed with the API key, bearer token, and signer
access code stripped.

## Handling credentials safely

- Treat every `signer-access-code` and signing URL as a credential. They grant
  access to the signing session; never log them. The SDK strips them from
  redirected requests and redacts them from network errors.
- Never ship an API key to a browser or mobile client. Keys are permanent and
  scoped to the user who created them. Never ask another Assinafy customer for
  theirs — that is what [OAuth](#connecting-other-peoples-workspaces-with-oauth)
  is for.
- An OAuth `client_secret` belongs only on a server you control, never in mobile
  code, browser code, or a repository. A public application is issued none and
  authenticates with PKCE. Rotating a lost secret invalidates the old one
  immediately, and deleting the application disconnects every user at once.
- `BaseURL` must be an absolute HTTP(S) URL with no embedded credentials, query,
  or fragment; anything else is rejected with `ErrInvalidBaseURL`.
- Downloaded artifacts are returned as bytes and are never written to disk by
  the SDK. Choose the path and permissions yourself — `0o600` for anything
  signer-identifying.
- Report a suspected vulnerability through [SECURITY.md](SECURITY.md).

## API coverage

[docs/API.md](docs/API.md) maps every operation to its SDK call, listing the
authentication mode, request fields, response payload, pagination or binary
behaviour, declared error codes, and a link to the official reference. It also
expands every response model field by field.

| Area | Operations | SDK resource |
| --- | ---: | --- |
| Accounts | 10 | `client.Accounts` |
| Assignments | 7 | `client.Assignments` |
| Authentication | 9 | `client.Authentication` |
| Documents | 18 | `client.Documents` |
| Fields | 8 | `client.Fields` |
| OAuth | 4 | the `oauth` package |
| Signers | 5 | `client.Signers` |
| Signing | 17 | `client.PublicDocuments`, `client.Signers`, `client.Assignments`, `client.SignerDocuments` |
| Tags | 4 | `client.Tags` |
| Templates | 1 | `client.Templates` |
| Users | 4 | `client.Users` |
| Webhooks | 6 | `client.Webhooks` |
| **Total** | **93** | |

Beyond those operations the SDK also exposes `Client.UploadAndRequestSignatures`,
`client.Templates.Get` (a single-template read the reference does not list as an
operation), `client.Authentication.SocialLoginURL`, the ICP-Brasil handshake, and
the webhook verifier. Deprecated methods retained for source compatibility are
listed at the end of [docs/API.md](docs/API.md#additional-exported-methods).

**Routes without a published contract.** `POST /v1/signers/certificate/start` and
`/complete` are named in the reference but declare no request body, response
payload, or error set. `client.Signers.StartCertificateSignature` and
`CompleteCertificateSignature` implement the shapes Assinafy's own signing flow
uses; treat them as unverified and see
[Signing with an ICP-Brasil certificate](#signing-with-an-icp-brasil-certificate).
Every other route reachable in production has a typed method here.

## Development

```bash
go build ./...
go test -race ./...
go vet ./...
gofmt -l .
golangci-lint run
```

GitHub Actions runs the same checks — plus `govulncheck` and a coverage floor —
on every push and pull request, using action revisions pinned by commit SHA. The
repository may be mirrored from GitLab; GitHub remains the CI execution target
shown by the badge above.

### Integration tests

`TestIntegration*` tests are skipped unless `ASSINAFY_RUN_INTEGRATION_TESTS=1`
and both `ASSINAFY_API_KEY` and `ASSINAFY_ACCOUNT_ID` are set. They default to
`https://sandbox.assinafy.com.br/v1` and refuse the production host unless
`ASSINAFY_RUN_PRODUCTION_TESTS=1` is also set.

The suite is not read-only: it creates, updates, and deletes temporary sandbox
resources. Use a disposable sandbox account. Two higher-impact flows need their
own opt-in.

| Variable | Purpose |
| --- | --- |
| `ASSINAFY_RUN_INTEGRATION_TESTS=1` | Enables the mutating integration suite. |
| `ASSINAFY_API_KEY` | Sandbox API key; required. |
| `ASSINAFY_ACCOUNT_ID` | Sandbox account ID; required. |
| `ASSINAFY_BASE_URL` | Optional override; empty uses `assinafy.SandboxBaseURL`. |
| `ASSINAFY_RUN_ASSIGNMENT_TESTS=1` | Runs the full assignment flow in a disposable account, sending real signature-request, public-token, and resend email. |
| `ASSINAFY_TEST_EMAIL_PRIMARY` | First runtime-only recipient; required with assignment tests. |
| `ASSINAFY_TEST_EMAIL_SECONDARY` | Second runtime-only recipient; required with assignment tests. |
| `ASSINAFY_RUN_ACCOUNT_ADMIN_TESTS=1` | Enables the account create/update/logo/delete lifecycle. |
| `ASSINAFY_RUN_PRODUCTION_TESTS=1` | Removes the production safety guard. It does not make the tests read-only. |

```bash
ASSINAFY_RUN_INTEGRATION_TESTS=1 ASSINAFY_API_KEY=... ASSINAFY_ACCOUNT_ID=... \
  go test -race -run '^TestIntegration' -v .
```

Enable production execution or the opt-in flows only when their side effects are
intended. Operations that would revoke the credential in use, change a password,
or fabricate a social-provider, password-reset, or signer-access token are
covered by local method, path, authentication, request, and response contract
tests instead of live calls.

## License

MIT — see [LICENSE](LICENSE).
