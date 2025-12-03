package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Term represents a single entry in the dictionary.
type Term struct {
	Term string `yaml:"term"`
	Yomi string `yaml:"yomi"`
	Ref  string `yaml:"ref,omitempty"`
}

// Dictionary represents the structure of the dictionary file.
type Dictionary struct {
	Terms []Term `yaml:"terms"`
}

// LoadDictionary loads and parses the dictionary file from the given path.
// It returns a map for efficient lookups.
func LoadDictionary(path string) (map[string]Term, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read dictionary file: %w", err)
	}

	var dict Dictionary
	if err := yaml.Unmarshal(data, &dict); err != nil {
		return nil, fmt.Errorf("failed to parse dictionary yaml: %w", err)
	}

	termMap := make(map[string]Term, len(dict.Terms))
	for _, term := range dict.Terms {
		if term.Term == "" || term.Yomi == "" {
			return nil, fmt.Errorf("invalid entry found: term and yomi are required")
		}
		if _, exists := termMap[term.Term]; exists {
			return nil, fmt.Errorf("duplicate term found: %s", term.Term)
		}
		termMap[term.Term] = term
	}

	return termMap, nil
}
