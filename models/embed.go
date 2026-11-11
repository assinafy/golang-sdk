package models

// EmbedOrigin is an exact origin allowed to embed a workspace's signing pages.
type EmbedOrigin struct {
	// ID identifies the registered origin.
	ID string `json:"id"`
	// Origin is the normalized scheme://host[:port], without a path.
	Origin string `json:"origin"`
	// CreatedAt is the origin's registration date-time.
	CreatedAt Timestamp `json:"created_at"`
}

// EmbedOriginRequest registers or replaces an allowed embedding origin.
type EmbedOriginRequest struct {
	// Origin is an exact HTTPS scheme://host[:port], without credentials,
	// path, query, fragment or wildcard. HTTP is allowed only for localhost
	// and 127.0.0.1. The API lowercases the host and drops a trailing slash
	// and the default port. A workspace can register at most ten origins.
	Origin string `json:"origin"`
}

// EmbedSession holds a single-use URL for an embedded signer.
type EmbedSession struct {
	// URL must be loaded in an iframe on an allowed origin within five
	// minutes. Opening it consumes its token; never log or persist this URL.
	URL string `json:"url"`
	// ExpiresAt is the session token's expiry date-time.
	ExpiresAt Timestamp `json:"expires_at"`
}

// ExchangeEmbedSessionRequest optionally identifies the page hosting the iframe.
// Assinafy's embed page performs this exchange; integrations load EmbedSession.URL.
type ExchangeEmbedSessionRequest struct {
	// ParentOrigin is the hosting page's origin recorded with the exchange.
	ParentOrigin string `json:"parent_origin,omitempty"`
}

// EmbedSessionExchange is the signing context returned to Assinafy's embed page.
type EmbedSessionExchange struct {
	// SignerAccessCode authenticates the embed page's signer requests. Keep it
	// secret; never log it or expose it to the parent page.
	SignerAccessCode string `json:"signer_access_code"`
	// SignerID identifies the signer whose session was exchanged.
	SignerID string `json:"signer_id"`
	// DocumentID identifies the document to sign.
	DocumentID string `json:"document_id"`
	// VerificationMethod is the signer's required identity verification method.
	VerificationMethod string `json:"verification_method"`
	// EmbedOrigins lists the exact origins permitted to host the signing page.
	EmbedOrigins []string `json:"embed_origins"`
}
