package oauth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const authorizationServerDocument = `{
  "issuer": "https://auth.assinafy.com.br",
  "authorization_endpoint": "https://auth.assinafy.com.br/oauth/authorize",
  "token_endpoint": "https://api.assinafy.com.br/v1/oauth/token",
  "revocation_endpoint": "https://api.assinafy.com.br/v1/oauth/revoke",
  "userinfo_endpoint": "https://api.assinafy.com.br/v1/oauth/userinfo",
  "jwks_uri": "https://auth.assinafy.com.br/.well-known/jwks.json",
  "scopes_supported": ["documents:read", "offline_access"],
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"],
  "token_endpoint_auth_methods_supported": ["client_secret_post", "none"],
  "authorization_response_iss_parameter_supported": true
}`

func TestDiscoverAuthorizationServer(t *testing.T) {
	var requested []string
	var issuer string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		if r.URL.Path != authorizationServerPath {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, strings.Replace(authorizationServerDocument, DefaultIssuer, issuer, 1))
	})

	issuer = srv.URL + "/"
	metadata, err := DiscoverAuthorizationServer(context.Background(), issuer, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(requested, []string{authorizationServerPath}) {
		t.Errorf("requested = %v", requested)
	}
	if metadata.Issuer != issuer || metadata.JWKSURI == "" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if !metadata.AuthorizationResponseIssParameterSupported {
		t.Error("iss parameter support was not decoded")
	}
	if !reflect.DeepEqual(metadata.CodeChallengeMethodsSupported, []string{ChallengeMethod}) {
		t.Errorf("code challenge methods = %v", metadata.CodeChallengeMethodsSupported)
	}

	endpoint := metadata.Endpoint()
	if endpoint.TokenURL != "https://api.assinafy.com.br/v1/oauth/token" || endpoint.Issuer != issuer {
		t.Fatalf("endpoint = %+v", endpoint)
	}
	// A Config built from discovery must not fall back to the defaults.
	if got := (Endpoint{}).resolve(); got != ProductionEndpoint() {
		t.Errorf("zero endpoint = %+v", got)
	}

	var nilMetadata *AuthorizationServerMetadata
	if got := nilMetadata.Endpoint(); got != (Endpoint{}) {
		t.Errorf("nil metadata endpoint = %+v", got)
	}
}

func TestDiscoverAuthorizationServerFallsBackToOpenIDConfiguration(t *testing.T) {
	var requested []string
	var issuer string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		if r.URL.Path != openIDConfigurationPath {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, strings.Replace(authorizationServerDocument, DefaultIssuer, issuer, 1))
	})

	issuer = srv.URL
	metadata, err := DiscoverAuthorizationServer(context.Background(), issuer, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Issuer != issuer {
		t.Fatalf("metadata = %+v", metadata)
	}
	if !reflect.DeepEqual(requested, []string{authorizationServerPath, openIDConfigurationPath}) {
		t.Errorf("requested = %v", requested)
	}
}

func TestDiscoverAuthorizationServerReportsTheOAuthFailure(t *testing.T) {
	srv := newTestServer(t, http.NotFound)
	if _, err := DiscoverAuthorizationServer(context.Background(), srv.URL, srv.Client()); err == nil {
		t.Fatal("expected an error when neither document exists")
	}
}

func TestDiscoverProtectedResource(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != protectedResourcePath {
			t.Errorf("path = %q", r.URL.Path)
		}
		// RFC 8615: the bare metadata object, never the API's envelope.
		_, _ = io.WriteString(w, `{"resource":"https://api.assinafy.com.br",
		  "authorization_servers":["https://auth.assinafy.com.br"],
		  "scopes_supported":["documents:read"],"bearer_methods_supported":["header"]}`)
	})

	metadata, err := DiscoverProtectedResource(context.Background(), srv.URL, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Resource != DefaultResource || metadata.Issuer() != DefaultIssuer {
		t.Fatalf("metadata = %+v", metadata)
	}
	if !reflect.DeepEqual(metadata.BearerMethodsSupported, []string{"header"}) {
		t.Errorf("bearer methods = %v", metadata.BearerMethodsSupported)
	}

	var nilMetadata *ProtectedResourceMetadata
	if nilMetadata.Issuer() != "" || (&ProtectedResourceMetadata{}).Issuer() != "" {
		t.Error("an empty document reported an issuer")
	}
}

func TestDiscoveryRejectsANonAbsoluteBase(t *testing.T) {
	if _, err := DiscoverProtectedResource(context.Background(), "not-a-url", nil); err == nil {
		t.Fatal("expected an error for a relative base")
	}
}

func TestDiscoveryDefaultsToProduction(t *testing.T) {
	// Both helpers accept an empty base and aim at production; a client that
	// refuses every request proves the target without making a network call.
	client := &http.Client{Transport: refusingTransport{}}
	if _, err := DiscoverAuthorizationServer(context.Background(), "", client); err == nil {
		t.Error("expected the refusing transport to fail the request")
	}
	if _, err := DiscoverProtectedResource(context.Background(), "", client); err == nil {
		t.Error("expected the refusing transport to fail the request")
	}
}

// refusingTransport asserts which host a default discovery call aims at without
// letting the request leave the machine.
type refusingTransport struct{}

func (refusingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	wantHost := "auth.assinafy.com.br"
	if r.URL.Path == protectedResourcePath {
		wantHost = "api.assinafy.com.br"
	}
	if r.URL.Host != wantHost {
		return nil, fmt.Errorf("discovery aimed at %q, want %q", r.URL.Host, wantHost)
	}
	return nil, errors.New("refused")
}

func TestDiscoveryRequiresTheRequestedIssuer(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		for _, document := range []string{`{}`, `{"issuer":"https://other.example"}`} {
			requests := 0
			srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if fallback && r.URL.Path == authorizationServerPath {
					http.NotFound(w, r)
					return
				}
				_, _ = io.WriteString(w, document)
			})
			got, err := DiscoverAuthorizationServer(context.Background(), srv.URL, srv.Client())
			if got != nil || !errors.Is(err, ErrIssuerMismatch) {
				t.Fatalf("fallback=%v result=%v error=%v", fallback, got, err)
			}
			want := 1
			if fallback {
				want = 2
			}
			if requests != want {
				t.Fatalf("requests=%d want=%d", requests, want)
			}
		}
	}
}
