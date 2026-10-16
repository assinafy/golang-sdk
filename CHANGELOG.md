# Changelog

All notable changes to the Assinafy Go SDK are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.11.0] - 2026-10-16

### Added

- Identity-document verification with `IdDocumentFaceMatchBasic`: signer status
  and consent-based capture sessions, account review summaries, evidence image
  downloads, and approval or rejection decisions, with typed request and response
  payloads. `StatusPendingIdentityReview` identifies documents awaiting review.
- `WebhookEndpoint.Secret` retains the signing secret returned when creating a
  signed endpoint. OAuth applications must store it from that response.
- `Webhooks.ListEndpointsByURL` and `DeleteEndpointByURL` select an endpoint by
  its exact registered URL.

### Fixed

- `UpdateWebhookEndpointRequest.Events` sends an empty non-nil slice as `[]` to
  clear the event selection; nil leaves it unchanged.
- Certificate verification documentation states the 0.5-credit signature fee
  plus the notification cost.

### Security

- OAuth discovery rejects a missing or mismatched metadata issuer with
  `ErrIssuerMismatch` before returning authorization-server endpoints.

## [1.10.1] - 2026-10-11

### Added

- `(*errors.APIError).RateLimit()` and `errors.ParseRateLimit` return the
  `X-Rate-Limit-Limit`, `X-Rate-Limit-Remaining` and `X-Rate-Limit-Reset` quota and
  the `Retry-After` delay of a `429 Too Many Requests` as `errors.RateLimit`.
  `oauth.Error` has the same `RateLimit()` method.
- `SignerDocuments.ListAsWorkspace` and `SignerDocuments.SearchAsWorkspace` read a
  signer's documents with the workspace API key or bearer token.
- `SignerDocuments.DownloadPublic` downloads a signer's document artifact with no
  credential, as the API contract declares for that operation.
- `models.Signer.GovernmentID` and `models.CreateSignerRequest.GovernmentID`
  (`government_id`, a CPF or CNPJ).
- `models.SendTokenChannelEmail`.

### Changed

- `PublicDocuments.SendToken`, `SendTokenWithRequest` and `SendTokenByEmail` send
  the required `{"channel": "email", "recipient": "<signer email>"}` body.
  `SendToken` returns the `{document, channel, recipient}` result. A nil body, an
  empty recipient or a channel other than `email` is rejected with
  `errors.ErrInvalidInput`. `SendDocumentTokenRequest.Email` is kept as an alias
  for `Recipient` and is no longer sent as an `email` property, which the API
  rejects.
- `Assignments.ResetExpirationWithRequest` always sends `expires_at`, which the API
  requires: a nil `ExpiresAt` sends `null` and removes the expiration.
  `ResetExpiration` behaves the same way and is no longer deprecated.
- `models.PublicDocumentInfo.PageCount` decodes `page_count` from a JSON integer,
  as the contract declares, or from a string.

### Security

- Every method that takes a signer access code sends only that code, never the API
  key or bearer token, and rejects an empty code with `errors.ErrInvalidInput`
  before any request is sent. This changes two behaviours:
  - `Fields.Validate` and `Fields.ValidateMultiple` with an empty code no longer
    fall back to the workspace credential; use `ValidateAuthenticated` and
    `ValidateMultipleAuthenticated` for workspace access.
  - `SignerDocuments.Download` requires a code; call `DownloadPublic` to download
    without one.
