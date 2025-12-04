package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// originalCmdRunner stores the original exec.Command function.
// It is used to mock the 'gh' command.
var originalCmdRunner = exec.Command

func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	os.Exit(code)
}

// Helper to create a temporary README.md with markers
func createTempReadme(t *testing.T, initialContent string) string {
	tmpfile, err := os.CreateTemp("", "README_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp README.md: %v", err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to temp README.md: %v", err)
	}
	return tmpfile.Name()
}

// TestFetchContributors tests fetching contributors from GitHub API
func TestFetchContributors(t *testing.T) {
	repo := "test_owner/test_repo"
	mockContributorsJSON := `[{"login": "contributor1", "html_url": "https://github.com/contributor1", "type": "User"},{"login": "bot-contributor", "html_url": "https://github.com/bot-contributor", "type": "Bot"},{"login": "contributor2", "html_url": "https://github.com/contributor2", "type": "User"}]`

	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" && arg[0] == "api" && strings.Contains(arg[len(arg)-1], "/contributors") {
			cmd := exec.Command("echo", mockContributorsJSON)
			return cmd
		}
		return originalCmdRunner(name, arg...)
	}
	defer func() { cmdRunner = oldCmdRunner }()

	contributors, err := fetchContributors(repo)
	if err != nil {
		t.Fatalf("fetchContributors() failed: %v", err)
	}

	if len(contributors) != 3 {
		t.Errorf("Expected 3 contributors, got %d", len(contributors))
	}
	if contributors[0].Login != "contributor1" || contributors[2].Login != "contributor2" {
		t.Errorf("Contributors not parsed correctly: %v", contributors)
	}
}

// TestFetchContributors_GhApiError tests error handling for gh api command failure
func TestFetchContributors_GhApiError(t *testing.T) {
	repo := "test_owner/test_repo"
	mockErrorStderr := "gh api error message"

	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" && arg[0] == "api" && strings.Contains(arg[len(arg)-1], "/contributors") {
			cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' >&2; exit 1", mockErrorStderr))
			return cmd
		}
		return originalCmdRunner(name, arg...)
	}
	defer func() { cmdRunner = oldCmdRunner }()

	_, err := fetchContributors(repo)
	if err == nil {
		t.Fatalf("fetchContributors() expected an error, got nil")
	}
	expectedErr := fmt.Sprintf("failed to fetch contributors from %s (gh api stderr: %s): exit status 1", repo, mockErrorStderr)
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("fetchContributors() error message = %q, want error message containing %q", err.Error(), expectedErr)
	}
}

// TestGenerateMarkdownList tests generating Markdown from contributors
func TestGenerateMarkdownList(t *testing.T) {
	contributors := []Contributor{
		{"contributor1", "https://github.com/contributor1", "User", false},
		{"bot-contributor", "https://github.com/bot-contributor", "Bot", false},
		{"contributor2", "https://github.com/contributor2", "User", false},
	}
	expectedMarkdown := "- [contributor1](https://github.com/contributor1)\n- [contributor2](https://github.com/contributor2)\n"

	markdown := generateMarkdownList(contributors)
	if markdown != expectedMarkdown {
		t.Errorf("generateMarkdownList() got = %q, want %q", markdown, expectedMarkdown)
	}
}

// TestUpdateReadme tests updating README.md with contributor list
func TestUpdateReadme(t *testing.T) {
	initialReadme := "\n# Project Title\n\n## 貢献者\n<!-- CONTRIBUTORS_START -->\n既存の貢献者リスト\n<!-- CONTRIBUTORS_END -->\n\n## 貢献ガイドライン\n"
	newContributorsMarkdown := "- [new_contributor](https://github.com/new_contributor)\n"
	expectedReadme := "\n# Project Title\n\n## 貢献者\n<!-- CONTRIBUTORS_START -->\n- [new_contributor](https://github.com/new_contributor)\n<!-- CONTRIBUTORS_END -->\n\n## 貢献ガイドライン\n"
	readmePath := createTempReadme(t, initialReadme)
	defer os.Remove(readmePath)

	err := updateReadme(readmePath, newContributorsMarkdown)
	if err != nil {
		t.Fatalf("updateReadme() failed: %v", err)
	}

	updatedContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read updated README.md: %v", err)
	}

	if string(updatedContent) != expectedReadme {
		t.Errorf("updateReadme() got = %q, want %q", string(updatedContent), expectedReadme)
	}
}

// TestUpdateReadme_NoMarkers tests updating README.md when markers are missing
func TestUpdateReadme_NoMarkers(t *testing.T) {
	initialReadme := "\n# Project Title\n## 貢献者\nここにリストが来るはず\n## 貢献ガイドライン\n"
	newContributorsMarkdown := "- [new_contributor](https://github.com/new_contributor)\n"
	readmePath := createTempReadme(t, initialReadme)
	defer os.Remove(readmePath)

	err := updateReadme(readmePath, newContributorsMarkdown)
	if err == nil {
		t.Fatalf("updateReadme() expected an error for missing markers, got nil")
	}
	if !strings.Contains(err.Error(), "CONTRIBUTORS_START or CONTRIBUTORS_END markers not found") {
		t.Errorf("updateReadme() error message = %q, want error message containing \"markers not found\"", err.Error())
	}
}

// TestRunEndToEnd tests the main function
func TestRunEndToEnd(t *testing.T) {
	repo := "test_owner/test_repo"
	mockContributorsJSON := `[{"login": "enduser1", "html_url": "https://github.com/enduser1", "type": "User"}]`
	initialReadme := "\n# Project Title\n\n## 貢献者\n<!-- CONTRIBUTORS_START -->\n<!-- CONTRIBUTORS_END -->\n"
	expectedFinalReadme := "\n# Project Title\n\n## 貢献者\n<!-- CONTRIBUTORS_START -->\n- [enduser1](https://github.com/enduser1)\n<!-- CONTRIBUTORS_END -->\n"
	
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

	testReadmePath := filepath.Join(tmpDir, "README.md")
	// Create a dummy README.md in the temp directory, as run() expects it to be at "README.md"
	err = os.WriteFile(testReadmePath, []byte(initialReadme), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial README.md: %v", err)
	}


	oldCmdRunner := cmdRunner
	cmdRunner = func(name string, arg ...string) *exec.Cmd {
		if name == "gh" && arg[0] == "api" && strings.Contains(arg[len(arg)-1], "/contributors") {
			cmd := exec.Command("echo", mockContributorsJSON)
			return cmd
		}
		return originalCmdRunner(name, arg...)
	}
	defer func() { cmdRunner = oldCmdRunner }()

	// Temporarily replace os.Args to simulate command line arguments
	oldArgs := os.Args
	os.Args = []string{"generate-contributors", repo}
	defer func() { os.Args = oldArgs }()

	err = run()
	if err != nil {
		t.Fatalf("run() failed: %v", err)
	}

	updatedContent, err := os.ReadFile(testReadmePath)
	if err != nil {
		t.Fatalf("Failed to read updated README.md: %v", err)
	}

	if string(updatedContent) != expectedFinalReadme {
		t.Errorf("run() updated README.md got = %q, want %q", string(updatedContent), expectedFinalReadme)
	}
}