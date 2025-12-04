package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// Helper function to create a temporary dictionary file
func createTempDictFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "test_dict_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	return tmpfile.Name()
}

func TestLoadDictionary(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantTermMap map[string]Term
		wantErr     bool
		errContains string
	}{
		{
			name: "valid YAML content",
			yamlContent: `
terms:
  - term: "Vite"
    yomi: "ヴィート"
    ref: "https://ja.vitejs.dev/"
  - term: "gRPC"
    yomi: "ジーアールピーシー"
`,
			wantTermMap: map[string]Term{
				"Vite": {"Vite", "ヴィート", "https://ja.vitejs.dev/"},
				"gRPC": {"gRPC", "ジーアールピーシー", ""}, // Ref is omitempty, so it's not set
			},
			wantErr: false,
		},
		{
			name: "invalid YAML syntax",
			yamlContent: `
terms:
  - term: "Vite"
    yomi: "ヴィート"
  - term: "gRPC"
    yomi: "ジーアールピーシー
`, // Missing quote
			wantTermMap: nil,
			wantErr:     true,
			errContains: "failed to parse dictionary yaml",
		},
		{
			name: "missing term field",
			yamlContent: `
terms:
  - yomi: "ヴィート"
`,
			wantTermMap: nil,
			wantErr:     true,
			errContains: "invalid entry found: term and yomi are required",
		},
		{
			name: "missing yomi field",
			yamlContent: `
terms:
  - term: "Vite"
`,
			wantTermMap: nil,
			wantErr:     true,
			errContains: "invalid entry found: term and yomi are required",
		},
		{
			name: "duplicate term",
			yamlContent: `
terms:
  - term: "Vite"
    yomi: "ヴィート"
  - term: "Vite"
    yomi: "ヴィート2"
`,
			wantTermMap: nil,
			wantErr:     true,
			errContains: "duplicate term found: Vite",
		},
		{
			name: "empty dictionary",
			yamlContent: `
terms: []
`,
			wantTermMap: make(map[string]Term),
			wantErr:     false,
		},
		{
			name:        "non-existent file",
			yamlContent: "", // Will be ignored, as we pass a non-existent path
			wantTermMap: nil,
			wantErr:     true,
			errContains: "failed to read dictionary file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.name == "non-existent file" {
				filePath = "non_existent_file.yaml"
			} else {
				filePath = createTempDictFile(t, tt.yamlContent)
				defer os.Remove(filePath)
			}

			got, err := LoadDictionary(filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadDictionary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("LoadDictionary() error message = %q, want error message containing %q", err.Error(), tt.errContains)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.wantTermMap) {
				t.Errorf("LoadDictionary() got = %v, want %v", got, tt.wantTermMap)
			}
		})
	}
}
