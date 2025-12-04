package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os" // Use os.ReadFile and os.WriteFile
	"os/exec"
	"strings"
)

// Contributor represents a GitHub contributor
type Contributor struct {
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"` // "User" or "Bot"
	SiteAdmin bool   `json:"site_admin"`
}

// cmdRunner is a package-level variable that can be overridden for testing.
var cmdRunner = exec.Command

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: %s <repository_owner/repository_name>", os.Args[0])
	}
	repo := os.Args[1] // e.g., takaryo1010/rubi

	contributors, err := fetchContributors(repo)
	if err != nil {
		return err
	}

	markdownList := generateMarkdownList(contributors)

	readmePath := "README.md"
	if err := updateReadme(readmePath, markdownList); err != nil {
		return err
	}

	fmt.Printf("Successfully updated %s with contributor list.\n", readmePath)
	return nil
}

func fetchContributors(repo string) ([]Contributor, error) {
	// Use gh api to fetch the contributors
	// gh api /repos/{owner}/{repo}/contributors
	cmd := cmdRunner("gh", "api", fmt.Sprintf("/repos/%s/contributors", repo))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("failed to fetch contributors from %s (gh api stderr: %s): %w", repo, strings.TrimSpace(stderr.String()), err)
		}
		return nil, fmt.Errorf("failed to fetch contributors from %s: %w", repo, err)
	}

	var contributors []Contributor
	if err := json.Unmarshal(stdout.Bytes(), &contributors); err != nil {
		return nil, fmt.Errorf("failed to parse contributors JSON: %w", err)
	}

	return contributors, nil
}

func generateMarkdownList(contributors []Contributor) string {
	var builder strings.Builder
	for _, c := range contributors {
		// Only list actual users, not bots
		if c.Type == "User" {
			builder.WriteString(fmt.Sprintf("- [%s](%s)\n", c.Login, c.HTMLURL))
		}
	}
	return builder.String()
}

func updateReadme(readmePath, contributorsMarkdown string) error {
	content, err := os.ReadFile(readmePath) // Use os.ReadFile
	if err != nil {
		return fmt.Errorf("failed to read README.md: %w", err)
	}

	startMarker := "<!-- CONTRIBUTORS_START -->"
	endMarker := "<!-- CONTRIBUTORS_END -->"

	startIndex := bytes.Index(content, []byte(startMarker))
	endIndex := bytes.Index(content, []byte(endMarker))

	if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
		return fmt.Errorf("CONTRIBUTORS_START or CONTRIBUTORS_END markers not found or are in invalid order in README.md")
	}

	var buffer bytes.Buffer
	buffer.Write(content[:startIndex+len(startMarker)]) // Write content before start marker
	buffer.WriteString("\n")
	buffer.WriteString(contributorsMarkdown)       // Write new contributors list
	buffer.WriteString(string(content[endIndex:])) // Write content from end marker onwards

	if err := os.WriteFile(readmePath, buffer.Bytes(), 0644); err != nil { // Use os.WriteFile
		return fmt.Errorf("failed to write updated README.md: %w", err)
	}

	return nil
}
