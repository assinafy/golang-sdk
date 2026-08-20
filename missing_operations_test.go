package assinafy

import "testing"

func TestNewClientInitializesUsers(t *testing.T) {
	client, err := NewClient(ClientOptions{BaseURL: "https://example.test"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.Users == nil {
		t.Fatal("Users resource is nil")
	}
}
