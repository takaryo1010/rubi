package main

import (
	"fmt"
	"os"
)

func main() {
	// 1. Check for command-line argument
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: Missing file path argument")
		os.Exit(1)
	}
	filepath := os.Args[1]

	// 2. Read the specified file
	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read file: %v\n", err)
		os.Exit(1)
	}

	// 3. Print the file's content to stdout
	fmt.Print(string(content))
}
