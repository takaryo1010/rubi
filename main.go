package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// 1. Define command-line flags
	var (
		dictPath   = flag.String("d", "dict.yaml", "Dictionary file path")
		write      = flag.Bool("w", false, "Write back to the file")
		scan       = flag.Bool("s", false, "Scan mode")
		check      = flag.Bool("c", false, "Check dictionary validity")
		dryRun     = flag.Bool("dry-run", false, "Dry run mode")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <input_file>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// TODO: Remove this block when the flags are actually used.
	// This is a temporary workaround to prevent "declared and not used" errors.
	_ = *dictPath
	_ = *scan
	_ = *check
	_ = *dryRun

	// 2. Check for the input file path
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	filepath := flag.Arg(0)

	// 3. Read the specified file
	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read file: %v\n", err)
		os.Exit(1)
	}

	// In this initial version, "processedContent" is the same as the original.
	processedContent := content

	// 4. Output the content
	if *write {
		// Write back to the file
		err := os.WriteFile(filepath, processedContent, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("File '%s' has been updated.\n", filepath)
	} else {
		// Print to stdout
		fmt.Print(string(processedContent))
	}
}
