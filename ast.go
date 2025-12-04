package main

import (
	"fmt"
	"os"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ProcessMarkdown parses the given Markdown content and traverses its AST.
// It applies the exclusion logic and logs encountered text nodes.
func ProcessMarkdown(content []byte, dryRun bool) ([]byte, error) {
	md := goldmark.New()
	document := md.Parser().Parse(text.NewReader(content))

	walker := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Implement exclusion logic
		switch n.Kind() {
		case ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindHTMLBlock, ast.KindRawHTML:
			if dryRun {
				fmt.Fprintf(os.Stderr, "Skipping %s node and its children (excluded scope).\n", n.Kind())
			}
			return ast.WalkSkipChildren, nil // Skip children of code/HTML blocks
		case ast.KindCodeSpan:
			if dryRun {
				fmt.Fprintf(os.Stderr, "Skipping %s node (inline code).\n", n.Kind())
			}
			return ast.WalkSkipChildren, nil // Skip children of inline code
		case ast.KindLink:
			if dryRun {
				fmt.Fprintf(os.Stderr, "Skipping %s node's URL part. Processing text content if any.\n", n.Kind())
			}
			return ast.WalkContinue, nil
		case ast.KindText:
			// Process text nodes
			segment := n.(*ast.Text).Segment
			text := segment.Value(content)

			// For now, just log the text.
			if dryRun {
				fmt.Fprintf(os.Stderr, "Found Text Node: '%s' (Offset: %d, Length: %d)\n", text, segment.Start, segment.Len())
			}
			return ast.WalkContinue, nil
		default:
			if dryRun {
				// Log other node types for debugging
				fmt.Fprintf(os.Stderr, "Encountered Node: %s\n", n.Kind())
			}
			return ast.WalkContinue, nil
		}
	}

	if err := ast.Walk(document, walker); err != nil {
		return nil, fmt.Errorf("error during AST traversal: %w", err)
	}

	// For this issue, we return the original content.
	// This will be replaced by actual patching later.
	return content, nil
}