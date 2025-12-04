package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// originalCmdRunner stores the original exec.Command function.
// It is intended to be used only by this test file to capture the real exec.Command.
// The actual mocking of 'cmdRunner' (defined in main.go) will happen per test.
var originalCmdRunner = exec.Command

func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	os.Exit(code)
}

// Helper to check if a file exists
func fileExists(t *testing.T, path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// --- Test handleInitCommand ---

func TestHandleInitCommand(t *testing.T) {
	repo := "owner/repo"
	testContent := "test: 123"

	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	// Mock cmdRunner for this test
	oldCmdRunner := cmdRunner // Save original
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" && arg[0] == "api" && strings.Contains(arg[len(arg)-1], "dict.yaml") {
			cmd := exec.Command("echo", testContent)
			return cmd
		}
		return originalCmdRunner(name, arg...) // Fallback to original for other commands
	}
	defer func() { cmdRunner = oldCmdRunner }() // Restore original cmdRunner after test

	tests := []struct {
		name        string
		overwrite   bool
		preExisting bool
		wantErr     bool
		errContains string
		wantFile    bool
	}{
		{
			name:        "successfully initialize new file",
			overwrite:   false,
			preExisting: false,
			wantErr:     false,
			wantFile:    true,
		},
		{
			name:        "file already exists, no overwrite",
			overwrite:   false,
			preExisting: true,
			wantErr:     true,
			errContains: "dict.yaml already exists",
			wantFile:    true, // file should still exist
		},
		{
			name:        "file already exists, with overwrite",
			overwrite:   true,
			preExisting: true,
			wantErr:     false,
			wantFile:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, "dict.yaml")
			if tt.preExisting {
				os.WriteFile(filePath, []byte("old content"), 0644)
			} else {
				os.Remove(filePath) // Ensure it doesn't exist
			}

			err := handleInitCommand(repo, tt.overwrite)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleInitCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("handleInitCommand() error message = %q, want error message containing %q", err.Error(), tt.errContains)
			}
			if tt.wantFile != fileExists(t, filePath) {
				t.Errorf("handleInitCommand() fileExists = %v, wantFile %v", fileExists(t, filePath), tt.wantFile)
			}
			if tt.wantFile && !tt.wantErr { // Check content only if file exists and no error
				content, _ := os.ReadFile(filePath)
				if strings.TrimSuffix(string(content), "\n") != testContent { // Trim newline from echo output
					t.Errorf("handleInitCommand() file content = %q, want %q", strings.TrimSuffix(string(content), "\n"), testContent)
				}
			}
		})
	}
}

// --- Test handleUpdateCommand ---

func TestHandleUpdateCommand(t *testing.T) {
	repo := "owner/repo"
	testContent := "updated: 456"

	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	// Mock cmdRunner for this test suite
	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" && arg[0] == "api" && strings.Contains(arg[len(arg)-1], "dict.yaml") {
			cmd := exec.Command("echo", testContent)
			return cmd
		}
		return originalCmdRunner(name, arg...)
	}
	defer func() { cmdRunner = oldCmdRunner }() // Restore original cmdRunner after test suite

	tests := []struct {
		name        string
		preExisting bool
		wantErr     bool
		errContains string
		wantFile    bool
	}{
		{
			name:        "successfully update existing file",
			preExisting: true,
			wantErr:     false,
			wantFile:    true,
		},
		{
			name:        "successfully update non-existing file",
			preExisting: false,
			wantErr:     false,
			wantFile:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, "dict.yaml")
			if tt.preExisting {
				os.WriteFile(filePath, []byte("old content"), 0644)
			} else {
				os.Remove(filePath)
			}

			err := handleUpdateCommand(repo)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleUpdateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("handleUpdateCommand() error message = %q, want error message containing %q", err.Error(), tt.errContains)
			}
			if tt.wantFile != fileExists(t, filePath) {
				t.Errorf("handleUpdateCommand() fileExists = %v, wantFile %v", fileExists(t, filePath), tt.wantFile)
			}
			if tt.wantFile && !tt.wantErr {
				content, _ := os.ReadFile(filePath)
				if strings.TrimSuffix(string(content), "\n") != testContent {
					t.Errorf("handleUpdateCommand() file content = %q, want %q", strings.TrimSuffix(string(content), "\n"), testContent)
				}
			}
		})
	}
}

// --- Test downloadDictFile ---

func TestDownloadDictFile(t *testing.T) {
	repo := "owner/repo"
	filePath := "test_dict.yaml"
	testContent := "downloaded: true"

	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	// Mock cmdRunner for this test
	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" {
			c := exec.Command("echo", testContent)
			return c
		}
		return originalCmdRunner(name, arg...)
	}
	defer func() { cmdRunner = oldCmdRunner }()

	err = downloadDictFile(repo, filePath)
	if err != nil {
		t.Fatalf("downloadDictFile() unexpectedly returned an error: %v", err)
	}

	if !fileExists(t, filePath) {
		t.Errorf("downloadDictFile() failed to create file")
	}

	content, _ := os.ReadFile(filePath)
	if strings.TrimSuffix(string(content), "\n") != testContent {
		t.Errorf("downloadDictFile() file content = %q, want %q", strings.TrimSuffix(string(content), "\n"), testContent)
	}
}

func TestDownloadDictFile_InvalidRepo(t *testing.T) {
	repo := "invalid-repo-format"
	filePath := "test_dict.yaml"

	// Mock cmdRunner for this test
	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		t.Fatalf("gh command should not be called for invalid repo format")
		return nil
	}
	defer func() { cmdRunner = oldCmdRunner }()

	err := downloadDictFile(repo, filePath)
	if err == nil {
		t.Errorf("downloadDictFile() expected an error for invalid repo, got nil")
	}
	if !strings.Contains(err.Error(), "invalid repository format") {
		t.Errorf("downloadDictFile() error message = %q, want error message containing \"invalid repository format\"", err.Error())
	}
}

func TestDownloadDictFile_GhApiError(t *testing.T) {
	repo := "owner/repo"
	filePath := "test_dict.yaml"

	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	// Mock cmdRunner to simulate a gh api error
	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" {
			c := exec.Command("bash", "-c", "echo 'gh api error' >&2; exit 1")
			return c
		}
		return originalCmdRunner(name, arg...)
	}
	defer func() { cmdRunner = oldCmdRunner }()

	err = downloadDictFile(repo, filePath)
	if err == nil {
		t.Errorf("downloadDictFile() expected an error for gh api failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to download dict.yaml from owner/repo (gh api stderr: gh api error): exit status 1") {
		t.Errorf("downloadDictFile() error message = %q, want error message containing \"failed to download dict.yaml from owner/repo (gh api stderr: gh api error): exit status 1\"", err.Error())
	}
}

func TestDownloadDictFile_WriteFileError(t *testing.T) {
	repo := "owner/repo"
	filePath := "nonexistent_dir/test_dict.yaml" // Path that will cause a write error
	testContent := "downloaded: true"

	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	// Mock cmdRunner to return predefined content
	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" {
			c := exec.Command("echo", testContent)
			return c
		}
		return originalCmdRunner(name, arg...)
	}
	defer func() { cmdRunner = oldCmdRunner }()

	err = downloadDictFile(repo, filePath)
	if err == nil {
		t.Errorf("downloadDictFile() expected an error for write file failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to write dict.yaml") {
		t.Errorf("downloadDictFile() error message = %q, want error message containing \"failed to write dict.yaml\"", err.Error())
	}
}
