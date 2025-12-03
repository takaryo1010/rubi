package main

import (
	"flag"
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	DictPath  string
	Write     bool
	Check     bool
	DryRun    bool
	InputFile string
}

func main() {
	// 1. Define and parse command-line flags
	dictPath := flag.String("d", "dict.yaml", "Dictionary file path")
	write := flag.Bool("w", false, "Write back to the file")
	check := flag.Bool("c", false, "Check dictionary validity")
	dryRun := flag.Bool("dry-run", false, "Dry run mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [input_file]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	cfg := &Config{
		DictPath: *dictPath,
		Write:    *write,
		Check:    *check,
		DryRun:   *dryRun,
	}

	// Get the input file path if not in check mode
	if !cfg.Check && flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	if flag.NArg() == 1 {
		cfg.InputFile = flag.Arg(0)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg *Config) error {
	// Handle --check mode
	if cfg.Check {
		return validateDictionary(cfg.DictPath)
	}

	// Load the dictionary
	termMap, err := LoadDictionary(cfg.DictPath)
	if err != nil {
		return err
	}
	// TODO: This is a placeholder to satisfy the "variable declared and not used" error.
	_ = termMap

	// Read the specified file
	content, err := os.ReadFile(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Process the Markdown content
	processedContent, err := ProcessMarkdown(content, cfg.DryRun)
	if err != nil {
		return fmt.Errorf("failed to process markdown: %w", err)
	}

	// Output the content
	if cfg.Write {
		if err := os.WriteFile(cfg.InputFile, processedContent, 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Printf("File '%s' has been updated.\n", cfg.InputFile)
	} else {
		fmt.Print(string(processedContent))
	}

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
