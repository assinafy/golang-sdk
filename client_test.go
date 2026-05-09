package assinafy

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name string
		opts ClientOptions
	}{
		{"with API key", ClientOptions{APIKey: "test-api-key", AccountID: "acc"}},
		{"with token", ClientOptions{Token: "test-token", AccountID: "acc"}},
		{"no credentials (public/auth flows only)", ClientOptions{AccountID: "acc"}},
		{"sandbox base URL", ClientOptions{APIKey: "k", BaseURL: SandboxBaseURL, AccountID: "acc"}},
		{"custom timeout", ClientOptions{APIKey: "k", Timeout: 60 * time.Second, AccountID: "acc"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClient(tc.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c == nil {
				t.Fatal("expected non-nil client")
			}
			if c.Documents == nil || c.Signers == nil || c.Assignments == nil ||
				c.Webhooks == nil || c.Templates == nil || c.Fields == nil ||
				c.Authentication == nil || c.PublicDocuments == nil || c.SignerDocuments == nil {
				t.Errorf("missing resource: %+v", c)
			}
		})
	}
}

func TestUploadAndRequestSignaturesValidatesSigners(t *testing.T) {
	c, err := NewClient(ClientOptions{APIKey: "k", AccountID: "acc"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UploadAndRequestSignatures(context.Background(), []byte("pdf"), "doc.pdf", nil, "", nil, "")
	if !errors.Is(err, ErrNoSigners) {
		t.Errorf("expected ErrNoSigners, got %v", err)
	}
}
