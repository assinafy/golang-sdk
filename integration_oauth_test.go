package assinafy

import (
	"context"
	"testing"
	"time"

	"github.com/assinafy/golang-sdk/oauth"
)

func TestIntegrationOAuthDiscovery(t *testing.T) {
	_, _ = integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server, err := oauth.DiscoverAuthorizationServer(ctx, oauth.DefaultIssuer, nil)
	if err != nil {
		t.Fatalf("DiscoverAuthorizationServer: %v", err)
	}
	if server.Issuer != oauth.DefaultIssuer || server.Endpoint() != oauth.ProductionEndpoint() || server.JWKSURI == "" {
		t.Errorf("unexpected public authorization-server metadata: %+v", server)
	}
	resource, err := oauth.DiscoverProtectedResource(ctx, oauth.DefaultResource, nil)
	if err != nil {
		t.Fatalf("DiscoverProtectedResource: %v", err)
	}
	if resource.Resource != oauth.DefaultResource || resource.Issuer() != oauth.DefaultIssuer {
		t.Errorf("unexpected public resource metadata: %+v", resource)
	}
}
