package assinafy

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    ClientOptions
		wantErr bool
	}{
		{
			name: "valid with API key",
			opts: ClientOptions{
				APIKey:    "test-api-key",
				AccountID: "test-account",
			},
			wantErr: false,
		},
		{
			name: "valid with token",
			opts: ClientOptions{
				Token:     "test-token",
				AccountID: "test-account",
			},
			wantErr: false,
		},
		{
			name: "invalid - no credentials",
			opts: ClientOptions{
				AccountID: "test-account",
			},
			wantErr: true,
		},
		{
			name: "custom base URL",
			opts: ClientOptions{
				APIKey:    "test-api-key",
				BaseURL:   "https://sandbox.assinafy.com.br/v1",
				AccountID: "test-account",
			},
			wantErr: false,
		},
		{
			name: "custom timeout",
			opts: ClientOptions{
				APIKey:    "test-api-key",
				Timeout:   60 * time.Second,
				AccountID: "test-account",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if client == nil {
				t.Error("expected client but got nil")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error' but got '%s'", err.Error())
	}
}
