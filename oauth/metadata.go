package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Well-known paths for the two discovery documents.
const (
	authorizationServerPath = "/.well-known/oauth-authorization-server"
	openIDConfigurationPath = "/.well-known/openid-configuration"
	protectedResourcePath   = "/.well-known/oauth-protected-resource"
)

// AuthorizationServerMetadata is the RFC 8414 document the authorization server
// publishes about itself. Most OAuth libraries need only the issuer and read
// the rest from here, so prefer discovery over hard-coding URLs.
type AuthorizationServerMetadata struct {
	// Issuer identifies the authorization server and must match the callback's iss.
	Issuer string `json:"issuer"`
	// AuthorizationEndpoint is the browser-facing approval page.
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	// TokenEndpoint exchanges codes and refresh tokens. It lives on the API host.
	TokenEndpoint string `json:"token_endpoint"`
	// RevocationEndpoint revokes an access or refresh token.
	RevocationEndpoint string `json:"revocation_endpoint"`
	// UserInfoEndpoint returns OpenID Connect claims.
	UserInfoEndpoint string `json:"userinfo_endpoint"`
	// JWKSURI serves the keys that sign id_tokens. Match a token's kid against them.
	JWKSURI string `json:"jwks_uri"`
	// ScopesSupported lists every scope the server can issue.
	ScopesSupported []string `json:"scopes_supported"`
	// ResponseTypesSupported lists the authorization response types, only "code".
	ResponseTypesSupported []string `json:"response_types_supported"`
	// GrantTypesSupported lists the token grants, authorization_code and refresh_token.
	GrantTypesSupported []string `json:"grant_types_supported"`
	// CodeChallengeMethodsSupported lists the PKCE methods, only "S256".
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
	// TokenEndpointAuthMethodsSupported lists how a client authenticates:
	// client_secret_post for confidential applications, none for public ones.
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	// AuthorizationResponseIssParameterSupported reports whether the callback
	// carries the iss parameter Config.ParseCallback validates.
	AuthorizationResponseIssParameterSupported bool `json:"authorization_response_iss_parameter_supported"`
}

// Endpoint converts the discovered metadata into a Config.Endpoint.
func (m *AuthorizationServerMetadata) Endpoint() Endpoint {
	if m == nil {
		return Endpoint{}
	}
	return Endpoint{
		Issuer:           m.Issuer,
		AuthorizationURL: m.AuthorizationEndpoint,
		TokenURL:         m.TokenEndpoint,
		RevocationURL:    m.RevocationEndpoint,
		UserInfoURL:      m.UserInfoEndpoint,
	}
}

// ProtectedResourceMetadata is the RFC 9728 document this API publishes about
// itself: which authorization servers can issue tokens for it and which scopes
// it accepts. It is also named by the WWW-Authenticate challenge on a 401 or 403.
type ProtectedResourceMetadata struct {
	// Resource is the RFC 8707 resource indicator identifying the API.
	Resource string `json:"resource"`
	// AuthorizationServers names the issuers that can mint tokens for it. Start
	// an integration by discovering the first entry.
	AuthorizationServers []string `json:"authorization_servers"`
	// ScopesSupported lists the scopes the API accepts. It deliberately omits
	// offline_access, which is a client concern rather than a protection.
	ScopesSupported []string `json:"scopes_supported"`
	// BearerMethodsSupported lists how a token may be presented, only "header".
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

// Issuer returns the first authorization server that can issue tokens for this
// resource, or "" when the document names none.
func (m *ProtectedResourceMetadata) Issuer() string {
	if m == nil || len(m.AuthorizationServers) == 0 {
		return ""
	}
	return m.AuthorizationServers[0]
}

// DiscoverAuthorizationServer fetches the RFC 8414 metadata for issuer, falling
// back to the OpenID Connect discovery document when the OAuth one is absent.
// Pass DefaultIssuer for production. A nil httpClient uses a 30-second default.
//
// The metadata and key documents are served only by the authorization server,
// never by the API host.
func DiscoverAuthorizationServer(ctx context.Context, issuer string, httpClient *http.Client) (*AuthorizationServerMetadata, error) {
	if issuer == "" {
		issuer = DefaultIssuer
	}
	cfg := &Config{HTTPClient: httpClient}

	var metadata AuthorizationServerMetadata
	err := fetchMetadata(ctx, cfg, issuer, authorizationServerPath, &metadata)
	if err != nil {
		if fallbackErr := fetchMetadata(ctx, cfg, issuer, openIDConfigurationPath, &metadata); fallbackErr == nil {
			return &metadata, nil
		}
		return nil, err
	}
	return &metadata, nil
}

// DiscoverProtectedResource fetches this API's RFC 9728 metadata. apiRoot is the
// API's origin without the version segment, for example
// https://api.assinafy.com.br; an empty value uses DefaultResource. A nil
// httpClient uses a 30-second default.
//
// GET /.well-known/oauth-protected-resource.
func DiscoverProtectedResource(ctx context.Context, apiRoot string, httpClient *http.Client) (*ProtectedResourceMetadata, error) {
	if apiRoot == "" {
		apiRoot = DefaultResource
	}
	var metadata ProtectedResourceMetadata
	if err := fetchMetadata(ctx, &Config{HTTPClient: httpClient}, apiRoot, protectedResourcePath, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// fetchMetadata reads a well-known document from the origin of base. Per RFC
// 8615 the response is the bare metadata object, never the API's envelope.
func fetchMetadata(ctx context.Context, cfg *Config, base, wellKnownPath string, result any) error {
	origin, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return fmt.Errorf("assinafy/oauth: %q is not an absolute URL", base)
	}
	origin.Path, origin.RawQuery, origin.Fragment = wellKnownPath, "", ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin.String(), nil)
	if err != nil {
		return fmt.Errorf("assinafy/oauth: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	return cfg.do(cfg.httpClient(), req, result)
}
