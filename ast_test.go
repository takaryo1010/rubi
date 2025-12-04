package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// Helper function to create a simplified termMap for testing
func createTestTermMap() map[string]Term {
	return map[string]Term{
		"Vite": {"Vite", "ヴィート", "https://ja.vitejs.dev/"},
		"gRPC": {"gRPC", "ジーアールピーシー", "https://grpc.io/"},
		"Go":   {"Go", "ゴー", ""},
		"golang": {"golang", "ゴーラング", ""},
	}
}

// Helper to compare []byte slices
func bytesEqual(a, b []byte) bool {
	return len(a) == len(b) && bytes.Equal(a, b)
}

// --- Test ApplyPatches ---

func TestApplyPatches(t *testing.T) {
	originalContent := []byte("This is a test with Vite and gRPC.")
	testTermMap := createTestTermMap()

	tests := []struct {
		name          string
		original      []byte
		patches       []Patch
		want          []byte
		wantErr       bool
		errContains   string
	}{
		{
			name:     "no patches",
			original: originalContent,
			patches:  []Patch{},
			want:     originalContent,
			wantErr:  false,
		},
		{
			name:     "single patch",
			original: []byte("Hello Vite"),
			patches: []Patch{
				{Start: 6, End: 10, NewText: []byte(fmt.Sprintf("<ruby>%s<rt>%s</rt></ruby>", "Vite", testTermMap["Vite"].Yomi))},
			},
			want:    []byte("Hello <ruby>Vite<rt>ヴィート</rt></ruby>"),
			wantErr: false,
		},
		{
			name:     "multiple non-overlapping patches (forward order)",
			original: []byte("Vite is great, gRPC is fast."),
			patches: []Patch{
				{Start: 0, End: 4, NewText: []byte(fmt.Sprintf("<ruby>%s<rt>%s</rt></ruby>", "Vite", testTermMap["Vite"].Yomi))},
				{Start: 15, End: 19, NewText: []byte(fmt.Sprintf("<ruby>%s<rt>%s</rt></ruby>", "gRPC", testTermMap["gRPC"].Yomi))},
			},
			want:    []byte("<ruby>Vite<rt>ヴィート</rt></ruby> is great, <ruby>gRPC<rt>ジーアールピーシー</rt></ruby> is fast."),
			wantErr: false,
		},
		{
			name:     "multiple non-overlapping patches (reverse order - should be sorted by func)",
			original: []byte("gRPC is fast, Vite is great."),
			patches: []Patch{
				{Start: 14, End: 18, NewText: []byte(fmt.Sprintf("<ruby>%s<rt>%s</rt></ruby>", "Vite", testTermMap["Vite"].Yomi))},
				{Start: 0, End: 4, NewText: []byte(fmt.Sprintf("<ruby>%s<rt>%s</rt></ruby>", "gRPC", testTermMap["gRPC"].Yomi))},
			},
			want:    []byte("<ruby>gRPC<rt>ジーアールピーシー</rt></ruby> is fast, <ruby>Vite<rt>ヴィート</rt></ruby> is great."),
			wantErr: false,
		},
		{
			name:     "overlapping patches should error",
			original: []byte("LongWordExample"),
			patches: []Patch{
				{Start: 0, End: 8, NewText: []byte("NewLong")},
				{Start: 5, End: 12, NewText: []byte("NewMid")},
			},
			want:        nil,
			wantErr:     true,
			errContains: "overlapping patches detected",
		},
		{
			name:     "out of bounds patch (start)",
			original: []byte("Hello"),
			patches: []Patch{
				{Start: -1, End: 3, NewText: []byte("Bad")},
			},
			want:        nil,
			wantErr:     true,
			errContains: "invalid patch bounds",
		},
		{
			name:     "out of bounds patch (end)",
			original: []byte("Hello"),
			patches: []Patch{
				{Start: 0, End: 6, NewText: []byte("Bad")},
			},
			want:        nil,
			wantErr:     true,
			errContains: "invalid patch bounds",
		},
		{
			name:     "patch resulting in shorter text",
			original: []byte("Hello world."),
			patches: []Patch{
				{Start: 6, End: 11, NewText: []byte("")}, // Remove "world"
			},
			want:    []byte("Hello ."),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyPatches(tt.original, tt.patches)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPatches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ApplyPatches() error message = %q, want error message containing %q", err.Error(), tt.errContains)
				}
			} else {
				if !bytesEqual(got, tt.want) {
					t.Errorf("ApplyPatches() got = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

// --- Test ProcessMarkdown (Manual Mode) ---

func TestProcessMarkdown_ManualMode(t *testing.T) {
	testTermMap := createTestTermMap()

	tests := []struct {
		name         string
		input        string
		dryRun       bool
		scan         bool
		firstOnly    bool
		wantOutput   string
		wantLogs     []string // Expected substrings in stderr output
		wantErr      bool
	}{
		{
			name:       "basic manual conversion",
			input:      "Hello Vite:rubi!",
			dryRun:     false,
			scan:       false,
			firstOnly:  false,
			wantOutput: "Hello <ruby>Vite<rt>ヴィート</rt></ruby>!",
			wantLogs:   nil,
		},
		{
			name:       "term not found",
			input:      "Hello Unknown:rubi!",
			dryRun:     false,
			scan:       false,
			firstOnly:  false,
			wantOutput: "Hello Unknown!",
			wantLogs:   []string{"WARNING (Manual Mode): Term 'Unknown' not found in dictionary. Removing ':rubi' suffix.\n"},
		},
		{
			name:       "multiple manual conversions",
			input:      "Vite:rubi is fast, gRPC:rubi is powerful.",
			dryRun:     false,
			scan:       false,
			firstOnly:  false,
			wantOutput: "<ruby>Vite<rt>ヴィート</rt></ruby> is fast, <ruby>gRPC<rt>ジーアールピーシー</rt></ruby> is powerful.",
			wantLogs:   nil,
		},
		{
			name:       "manual mode - code block ignored",
			input:      "```go\nVite:rubi\n```\nOutside: Vite:rubi",
			dryRun:     false,
			scan:       false,
			firstOnly:  false,
			wantOutput: "```go\nVite:rubi\n```\nOutside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "manual mode - inline code ignored",
			input:      "This `Vite:rubi` is good. Outside: Vite:rubi",
			dryRun:     false,
			scan:       false,
			firstOnly:  false,
			wantOutput: "This `Vite:rubi` is good. Outside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "manual mode - link text processed (URL part ignored)",
			input:      "[Vite:rubi](http://example.com/Vite:rubi) Outside: Vite:rubi",
			dryRun:     false,
			scan:       false,
			firstOnly:  false,
			wantOutput: "[<ruby>Vite<rt>ヴィート</rt></ruby>](http://example.com/Vite:rubi) Outside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "manual mode - HTML block processed outside (separated by blank line)",
			input:      "<div>Vite:rubi</div>\n\nOutside: Vite:rubi", // Added blank line
			dryRun:     false,
			scan:       false,
			firstOnly:  false,
			wantOutput: "<div>Vite:rubi</div>\n\nOutside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "manual mode - dry run",
			input:      "Hello Vite:rubi!",
			dryRun:     true,
			scan:       false,
			firstOnly:  false,
			wantOutput: "Hello Vite:rubi!", // Original content
			wantLogs:   []string{"GENERATING PATCH (Manual Mode): Found 'Vite:rubi', replace with '<ruby>Vite<rt>ヴィート</rt></ruby>' (Offset: 6-15)\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			got, err := ProcessMarkdown([]byte(tt.input), tt.dryRun, tt.scan, tt.firstOnly, testTermMap)

			w.Close()
			os.Stderr = oldStderr // Restore stderr
			
			// Read captured stderr
			stderrBytes, _ := io.ReadAll(r)
			stderrOutput := string(stderrBytes)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessMarkdown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if len(tt.wantLogs) == 0 { // Check if wantLogs is empty when wantErr is true
					t.Errorf("Test case error: wantErr is true but wantLogs is nil or empty")
					return
				}
				if !strings.Contains(err.Error(), tt.wantLogs[0]) { // Assuming first log is error msg
					t.Errorf("ProcessMarkdown() error message = %q, want error message containing %q", err.Error(), tt.wantLogs[0])
				}
			} else {
				if !bytesEqual(got, []byte(tt.wantOutput)) {
					t.Errorf("ProcessMarkdown() got = %q, want %q", got, tt.wantOutput)
				}
				for _, log := range tt.wantLogs {
					if !strings.Contains(stderrOutput, log) {
						t.Errorf("ProcessMarkdown() stderr output = %q, want log containing %q", stderrOutput, log)
					}
				}
			}
		})
	}
}

// --- Test ProcessMarkdown (Scan Mode) ---

func TestProcessMarkdown_ScanMode(t *testing.T) {
	testTermMap := createTestTermMap()

	tests := []struct {
		name         string
		input        string
		dryRun       bool
		scan         bool
		firstOnly    bool
		wantOutput   string
		wantLogs     []string
		wantErr      bool
	}{
		{
			name:       "basic scan conversion",
			input:      "Hello Vite, this is gRPC.",
			dryRun:     false,
			scan:       true,
			firstOnly:  false,
			wantOutput: "Hello <ruby>Vite<rt>ヴィート</rt></ruby>, this is <ruby>gRPC<rt>ジーアールピーシー</rt></ruby>.",
			wantLogs:   nil,
		},
		{
			name:       "scan mode - multiple occurrences",
			input:      "Vite is good. Vite is fast. gRPC is also good.",
			dryRun:     false,
			scan:       true,
			firstOnly:  false,
			wantOutput: "<ruby>Vite<rt>ヴィート</rt></ruby> is good. <ruby>Vite<rt>ヴィート</rt></ruby> is fast. <ruby>gRPC<rt>ジーアールピーシー</rt></ruby> is also good.",
			wantLogs:   nil,
		},
		{
			name:       "scan mode - first only",
			input:      "Vite is good. Vite is fast. gRPC is also good.",
			dryRun:     false,
			scan:       true,
			firstOnly:  true,
			wantOutput: "<ruby>Vite<rt>ヴィート</rt></ruby> is good. Vite is fast. <ruby>gRPC<rt>ジーアールピーシー</rt></ruby> is also good.",
			wantLogs:   nil,
		},
		{
			name:       "scan mode - code block ignored",
			input:      "```go\nVite\n```\nOutside: Vite",
			dryRun:     false,
			scan:       true,
			firstOnly:  false,
			wantOutput: "```go\nVite\n```\nOutside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "scan mode - inline code ignored",
			input:      "This `Vite` is good. Outside: Vite",
			dryRun:     false,
			scan:       true,
			firstOnly:  false,
			wantOutput: "This `Vite` is good. Outside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "scan mode - link text processed",
			input:      "[Vite](http://example.com/Vite) Outside: Vite",
			dryRun:     false,
			scan:       true,
			firstOnly:  false,
			wantOutput: "[<ruby>Vite<rt>ヴィート</rt></ruby>](http://example.com/Vite) Outside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "scan mode - HTML block processed outside (separated by blank line)",
			input:      "<div>Vite</div>\n\nOutside: Vite", // Added blank line
			dryRun:     false,
			scan:       true,
			firstOnly:  false,
			wantOutput: "<div>Vite</div>\n\nOutside: <ruby>Vite<rt>ヴィート</rt></ruby>",
			wantLogs:   nil,
		},
		{
			name:       "scan mode - dry run",
			input:      "Hello Vite!",
			dryRun:     true,
			scan:       true,
			firstOnly:  false,
			wantOutput: "Hello Vite!", // Original content
			wantLogs:   []string{"GENERATING PATCH (Scan Mode): Found 'Vite', replace with '<ruby>Vite<rt>ヴィート</rt></ruby>' (Offset: 6-10)\n"},
		},
		{
			name:       "scan mode - dry run with first only",
			input:      "Vite is good. Vite is fast.",
			dryRun:     true,
			scan:       true,
			firstOnly:  true,
			wantOutput: "Vite is good. Vite is fast.", // Original content
			wantLogs:   []string{"GENERATING PATCH (Scan Mode): Found 'Vite', replace with '<ruby>Vite<rt>ヴィート</rt></ruby>' (Offset: 0-4)\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			got, err := ProcessMarkdown([]byte(tt.input), tt.dryRun, tt.scan, tt.firstOnly, testTermMap)

			w.Close()
			os.Stderr = oldStderr // Restore stderr
			
			// Read captured stderr
			stderrBytes, _ := io.ReadAll(r)
			stderrOutput := string(stderrBytes)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessMarkdown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if len(tt.wantLogs) == 0 { // Check if wantLogs is empty when wantErr is true
					t.Errorf("Test case error: wantErr is true but wantLogs is nil or empty")
					return
				}
				if !strings.Contains(err.Error(), tt.wantLogs[0]) {
					t.Errorf("ProcessMarkdown() error message = %q, want error message containing %q", err.Error(), tt.wantLogs[0])
				}
			} else {
				if !bytesEqual(got, []byte(tt.wantOutput)) {
					t.Errorf("ProcessMarkdown() got = %q, want %q", got, tt.wantOutput)
				}
				for _, log := range tt.wantLogs {
					if !strings.Contains(stderrOutput, log) {
						t.Errorf("ProcessMarkdown() stderr output = %q, want log containing %q", stderrOutput, log)
					}
				}
			}
		})
	}
}
