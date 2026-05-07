package models

import (
	"testing"
)

func TestListParams_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    ListParams
		expected ListParams
	}{
		{
			name:  "zero values",
			input: ListParams{},
			expected: ListParams{
				Page:    1,
				PerPage: 25,
			},
		},
		{
			name: "custom page",
			input: ListParams{
				Page: 5,
			},
			expected: ListParams{
				Page:    5,
				PerPage: 25,
			},
		},
		{
			name: "custom per page",
			input: ListParams{
				PerPage: 50,
			},
			expected: ListParams{
				Page:    1,
				PerPage: 50,
			},
		},
		{
			name: "per page over 100",
			input: ListParams{
				PerPage: 150,
			},
			expected: ListParams{
				Page:    1,
				PerPage: 100,
			},
		},
		{
			name: "all values set",
			input: ListParams{
				Page:    2,
				PerPage: 50,
				Search:  "test",
				Sort:    "-created_at",
			},
			expected: ListParams{
				Page:    2,
				PerPage: 50,
				Search:  "test",
				Sort:    "-created_at",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.SetDefaults()
			if tt.input.Page != tt.expected.Page {
				t.Errorf("Page: expected %d but got %d", tt.expected.Page, tt.input.Page)
			}
			if tt.input.PerPage != tt.expected.PerPage {
				t.Errorf("PerPage: expected %d but got %d", tt.expected.PerPage, tt.input.PerPage)
			}
			if tt.input.Search != tt.expected.Search {
				t.Errorf("Search: expected %s but got %s", tt.expected.Search, tt.input.Search)
			}
			if tt.input.Sort != tt.expected.Sort {
				t.Errorf("Sort: expected %s but got %s", tt.expected.Sort, tt.input.Sort)
			}
		})
	}
}

func TestDocumentStatus(t *testing.T) {
	statuses := []DocumentStatus{
		StatusUploading,
		StatusUploaded,
		StatusMetadataProcessing,
		StatusMetadataReady,
		StatusExpired,
		StatusCertificating,
		StatusCertificated,
		StatusRejectedBySigner,
		StatusPendingSignature,
		StatusRejectedByUser,
		StatusFailed,
	}

	expected := []string{
		"uploading",
		"uploaded",
		"metadata_processing",
		"metadata_ready",
		"expired",
		"certificating",
		"certificated",
		"rejected_by_signer",
		"pending_signature",
		"rejected_by_user",
		"failed",
	}

	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("expected %s but got %s", expected[i], status)
		}
	}
}

func TestAssignmentMethod(t *testing.T) {
	if string(MethodVirtual) != "virtual" {
		t.Errorf("expected 'virtual' but got '%s'", MethodVirtual)
	}
	if string(MethodCollect) != "collect" {
		t.Errorf("expected 'collect' but got '%s'", MethodCollect)
	}
}
