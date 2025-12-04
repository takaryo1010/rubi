package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Config holds the application configuration
type Config struct {
	DictPath  string
	Write     bool
	Scan      bool // Reintroduced scan flag
	FirstOnly bool // New first-only flag
	Check     bool
	DryRun    bool
	InputFile string
}

// Global flags for the main command
var (
	mainFlagSet = flag.NewFlagSet("rubi", flag.ExitOnError)
	dictPath    = mainFlagSet.String("d", "dict.yaml", "Dictionary file path")
	write       = mainFlagSet.Bool("w", false, "Write back to the file")
	scan        = mainFlagSet.Bool("s", false, "Scan mode")
	firstOnly   = mainFlagSet.Bool("first-only", false, "Convert only the first occurrence of each term in scan mode")
	check       = mainFlagSet.Bool("c", false, "Check dictionary validity")
	dryRun      = mainFlagSet.Bool("dry-run", false, "Dry run mode")
)

// cmdRunner is a package-level variable that can be overridden for testing.
var cmdRunner = exec.Command

// Subcommand flag sets
var (
	initFlagSet   = flag.NewFlagSet("init", flag.ExitOnError)
	initRepo      = initFlagSet.String("repo", "takaryo1010/rubi", "GitHub repository to download dict.yaml from (e.g., owner/repo)")
	initOverwrite = initFlagSet.Bool("overwrite", false, "Overwrite existing dict.yaml if it exists")

	dictUpdateFlagSet = flag.NewFlagSet("dict update", flag.ExitOnError) // FlagSet for 'dict update'
	dictUpdateRepo    = dictUpdateFlagSet.String("repo", "takaryo1010/rubi", "GitHub repository to download dict.yaml from (e.g., owner/repo)")
)

func main() {
	if err := runCLI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCLI() error {
	// Custom usage function for the main command
	mainFlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [options] <input_file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s <command> [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  init        Initialize a dict.yaml from GitHub\n")
		fmt.Fprintf(os.Stderr, "  dict update Update dict.yaml from GitHub\n") // Updated usage
		fmt.Fprintf(os.Stderr, "Options for main command:\n")
		mainFlagSet.PrintDefaults()
	}

	// Determine subcommand or main command
	args := os.Args[1:]
	if len(args) == 0 { // No arguments, show main usage
		mainFlagSet.Usage()
		return nil // Exit cleanly after showing usage
	}

	// Check if the first non-flag argument is a known subcommand
	// We need to peek ahead for subcommands because mainFlagSet might consume it.
	subcommand := ""
	if !strings.HasPrefix(args[0], "-") { // If first arg is not a flag, it might be a subcommand
		switch args[0] {
		case "init", "dict", "help":
			subcommand = args[0]
		}
	}

	if subcommand != "" {
		// Dispatch to subcommand handlers
		switch subcommand {
		case "init":
			initFlagSet.Usage = func() {
				fmt.Fprintf(os.Stderr, "Usage of %s init:\n", os.Args[0])
				fmt.Fprintf(os.Stderr, "  %s init [options]\n", os.Args[0])
				initFlagSet.PrintDefaults()
			}
			initFlagSet.Parse(args[1:])
			return handleInitCommand(*initRepo, *initOverwrite)
		case "dict":
			if len(args) < 2 {
				return fmt.Errorf("missing subcommand for 'dict'\n\nUsage: %s dict <command> [options]\nCommands:\n  update", os.Args[0])
			}
			switch args[1] { // Check second argument for 'dict' subcommand
			case "update":
				dictUpdateFlagSet.Usage = func() {
					fmt.Fprintf(os.Stderr, "Usage of %s dict update:\n", os.Args[0])
					fmt.Fprintf(os.Stderr, "  %s dict update [options]\n", os.Args[0])
					dictUpdateFlagSet.PrintDefaults()
				}
				dictUpdateFlagSet.Parse(args[2:])
				return handleUpdateCommand(*dictUpdateRepo)
			default:
				return fmt.Errorf("unknown subcommand for 'dict': %s\n\nUsage: %s dict <command> [options]\nCommands:\n  update", args[1], os.Args[0])
			}
		case "help":
			mainFlagSet.Usage()
			return nil
		default:
			return fmt.Errorf("unknown command: %s", args[0]) // Should not be reached due to subcommand check
		}
	} else {
		// No subcommand found, treat all arguments as belonging to the main command
		mainFlagSet.Parse(args)
		cfg := &Config{
			DictPath:  *dictPath,
			Write:     *write,
			Scan:      *scan,
			FirstOnly: *firstOnly,
			Check:     *check,
			DryRun:    *dryRun,
		}
		if mainFlagSet.NArg() > 0 {
			cfg.InputFile = mainFlagSet.Arg(0)
		}
		return handleMainCommand(cfg)
	}
}

func handleMainCommand(cfg *Config) error {
	// Logic for flag validation (from previous iteration)
	if cfg.Check {
		if cfg.InputFile != "" {
			mainFlagSet.Usage()
			return fmt.Errorf("the -c flag cannot be used with an input file")
		}
		if cfg.Scan || cfg.FirstOnly || cfg.Write || cfg.DryRun {
			return fmt.Errorf("the -c flag cannot be used with other processing flags (-s, --first-only, -w, --dry-run)")
		}
	} else if cfg.Scan { // Scan mode validation
		if cfg.InputFile == "" {
			mainFlagSet.Usage()
			return fmt.Errorf("an input file is required for scan mode (-s)")
		}
	} else { // Manual mode validation
		if cfg.FirstOnly {
			return fmt.Errorf("the --first-only flag is only valid in -s (scan) mode")
		}
		if cfg.InputFile == "" {
			mainFlagSet.Usage()
			return fmt.Errorf("an input file is required for manual mode")
		}
	}

	// Handle --check mode
	if cfg.Check {
		return validateDictionary(cfg.DictPath)
	}

	// Load the dictionary
	termMap, err := LoadDictionary(cfg.DictPath)
	if err != nil {
		return err
	}

	// Read the specified file
	content, err := os.ReadFile(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read file '%s': %w", cfg.InputFile, err)
	}

	// Process the Markdown content
	processedContent, err := ProcessMarkdown(content, cfg.DryRun, cfg.Scan, cfg.FirstOnly, termMap)
	if err != nil {
		return fmt.Errorf("failed to process markdown: %w", err)
	}

	// Output the content
	if cfg.Write {
		if err := os.WriteFile(cfg.InputFile, processedContent, 0644); err != nil {
			return fmt.Errorf("failed to write to file '%s': %w", cfg.InputFile, err)
		}
		fmt.Printf("File '%s' has been updated.\n", cfg.InputFile)
	} else {
		fmt.Print(string(processedContent))
	}

	return nil
}

func handleInitCommand(repo string, overwrite bool) error {
	fmt.Printf("Initializing dict.yaml from %s...\n", repo)
	filePath := "dict.yaml"

	if _, err := os.Stat(filePath); err == nil {
		if !overwrite {
			return fmt.Errorf("dict.yaml already exists. Use --overwrite to replace it.")
		}
	}

	return downloadDictFile(repo, filePath)
}

func handleUpdateCommand(repo string) error {
	fmt.Printf("Updating dict.yaml from %s...\n", repo)
	filePath := "dict.yaml"
	return downloadDictFile(repo, filePath)
}

func downloadDictFile(repo, filePath string) error {
	// Construct the GitHub API URL for the raw file content
	// Assuming dict.yaml is at the root of the main branch
	ownerRepo := strings.Split(repo, "/")
	if len(ownerRepo) != 2 {
		return fmt.Errorf("invalid repository format: %s. Expected owner/repo", repo)
	}
	owner := ownerRepo[0]
	repoName := ownerRepo[1]
	
	// Use gh api to fetch the raw content
	// gh api -H "Accept: application/vnd.github.v3.raw" /repos/{owner}/{repo}/contents/dict.yaml
	cmd := cmdRunner("gh", "api", "-H", "Accept: application/vnd.github.v3.raw", fmt.Sprintf("/repos/%s/%s/contents/dict.yaml", owner, repoName))
	
	// Capture stdout
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include gh cli stderr in the returned error for better debugging
		if stderr.Len() > 0 {
			return fmt.Errorf("failed to download dict.yaml from %s (gh api stderr: %s): %w", repo, strings.TrimSpace(stderr.String()), err)
		}
		return fmt.Errorf("failed to download dict.yaml from %s: %w", repo, err)
	}

	// Write the content to file
	if err := os.WriteFile(filePath, stdout.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write dict.yaml: %w", err)
	}

	fmt.Printf("Successfully downloaded dict.yaml to %s.\n", filePath)
	return nil
}

// validateDictionary performs validation on the dictionary file.
func validateDictionary(path string) error {
	_, err := LoadDictionary(path)
	if err != nil {
		return fmt.Errorf("dictionary validation failed: %w", err)
	}
	fmt.Printf("Dictionary at '%s' is valid.\n", path)
	return nil
}