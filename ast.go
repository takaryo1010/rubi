package main

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"regexp"
	"sort"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Regex to find "word:rubi". It's a package-level variable to avoid recompilation.
// Matches one or more word characters (alphanumeric and underscore).
// Note: \w in Go's regex typically includes alphanumeric and underscore.
// For Japanese characters, a different regex might be needed (e.g., Unicode categories).
var rubiRegex = regexp.MustCompile(`(\w+):rubi\b`)

// Patch represents a single change to be applied to the content.
type Patch struct {
	Start   int
	End     int
	NewText []byte
}

// ApplyPatches applies a list of patches to the original content.
// It sorts patches, validates them, and builds the new content in a single pass.
func ApplyPatches(original []byte, patches []Patch) ([]byte, error) {
	// Sort patches in forward order by start offset
	sort.Slice(patches, func(i, j int) bool {
		return patches[i].Start < patches[j].Start
	})

	// Validate patches
	for i := 0; i < len(patches); i++ {
		p := patches[i]
		if p.Start < 0 || p.End > len(original) || p.Start > p.End { // Corrected validation
			return nil, fmt.Errorf("invalid patch bounds: %+v, original length: %d", p, len(original))
		}
		// Check for overlapping patches (only if not the first patch)
		if i > 0 && p.Start < patches[i-1].End {
			return nil, fmt.Errorf("overlapping patches detected: patch at %d-%d overlaps with patch at %d-%d",
				patches[i-1].Start, patches[i-1].End, p.Start, p.End)
		}
	}

	var buf bytes.Buffer
	lastIndex := 0
	for _, p := range patches {
		// Write content before this patch
		buf.Write(original[lastIndex:p.Start])
		// Write the new text for the patch
		buf.Write(p.NewText)
		lastIndex = p.End
	}

	// Write any remaining original content after the last patch
	buf.Write(original[lastIndex:])

	return buf.Bytes(), nil
}

// ProcessMarkdown parses the given Markdown content and traverses its AST.
// In manual mode, it finds words marked with the ":rubi" suffix and converts them to HTML ruby tags.
// In scan mode, it automatically detects all dictionary terms and converts them to HTML ruby tags.
// The firstOnly parameter (only valid in scan mode) limits conversion to the first occurrence of each term.
// All conversions are based on the provided term dictionary.
func ProcessMarkdown(content []byte, dryRun bool, scan bool, firstOnly bool, termMap map[string]Term) ([]byte, error) {
	md := goldmark.New()
	document := md.Parser().Parse(text.NewReader(content))

	var patches []Patch
	// To track terms for firstOnly. Note: this tracking is case-sensitive based on matched word.
	// If case-insensitivity is desired, terms should be normalized (e.g., to lowercase) before tracking.
	processedTerms := make(map[string]bool) 

	walker := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Implement exclusion logic
		switch n.Kind() {
		case ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindHTMLBlock, ast.KindRawHTML, ast.KindCodeSpan:
			return ast.WalkSkipChildren, nil
		case ast.KindLink:
			return ast.WalkContinue, nil
		case ast.KindText:
			segment := n.(*ast.Text).Segment
			textBytes := segment.Value(content)
			textStr := string(textBytes)

			if scan {
				// Scan mode: find any dictionary term
				for termStr, termData := range termMap {
					// Use a word boundary to prevent partial matches (e.g., "go" matching "golang")
					// This regex finds all occurrences of the term in the text node
					// We use a non-capturing group for the word boundary \b to avoid issues with FindAllStringSubmatchIndex
					// Note: \w in Go's regex typically includes alphanumeric and underscore.
					// For Japanese characters, a different regex might be needed (e.g., Unicode categories).
					scanTermRegex := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(termStr)))
					matches := scanTermRegex.FindAllStringSubmatchIndex(textStr, -1)

					for _, match := range matches {
						word := textStr[match[2]:match[3]] // Extract the matched word (first capturing group)

						// Check if term already processed in firstOnly mode. Case-sensitive tracking based on matched word.
						if firstOnly && processedTerms[word] {
							continue
						}

						fullMatchStart := segment.Start + match[0]
						fullMatchEnd := segment.Start + match[1]

						// Term found in dictionary, create ruby tag, escaping HTML characters
						safeWord := html.EscapeString(word)
						safeYomi := html.EscapeString(termData.Yomi)
						newText := fmt.Sprintf("<ruby>%s<rt>%s</rt></ruby>", safeWord, safeYomi)
						patches = append(patches, Patch{Start: fullMatchStart, End: fullMatchEnd, NewText: []byte(newText)})
						if dryRun {
							fmt.Fprintf(os.Stderr, "GENERATING PATCH (Scan Mode): Found '%s', replace with '%s' (Offset: %d-%d)\n", word, newText, fullMatchStart, fullMatchEnd)
						}
						processedTerms[word] = true // Mark as processed
					}
				}
			} else {
				// Manual mode: find "word:rubi"
				matches := rubiRegex.FindAllStringSubmatchIndex(textStr, -1)
				if len(matches) == 0 {
					return ast.WalkContinue, nil
				}

				for _, match := range matches {
					fullMatchStart := segment.Start + match[0]
					fullMatchEnd := segment.Start + match[1]
					wordStart := segment.Start + match[2]
					wordEnd := segment.Start + match[3]
					
					originalWordStr := string(content[wordStart:wordEnd])

					if term, found := termMap[originalWordStr]; found {
						// Term found in dictionary, create ruby tag, escaping HTML characters
						safeWord := html.EscapeString(originalWordStr)
						safeYomi := html.EscapeString(term.Yomi)
						newText := fmt.Sprintf("<ruby>%s<rt>%s</rt></ruby>", safeWord, safeYomi)
						patches = append(patches, Patch{Start: fullMatchStart, End: fullMatchEnd, NewText: []byte(newText)})
						if dryRun {
							fmt.Fprintf(os.Stderr, "GENERATING PATCH (Manual Mode): Found '%s:rubi', replace with '%s' (Offset: %d-%d)\n", originalWordStr, newText, fullMatchStart, fullMatchEnd)
						}
					} else {
						// Term not found, remove ":rubi" suffix
						patches = append(patches, Patch{Start: wordEnd, End: fullMatchEnd, NewText: []byte("")})
						if dryRun {
							fmt.Fprintf(os.Stderr, "WARNING (Manual Mode): Term '%s' not found in dictionary. The ':rubi' suffix would be removed (dry-run mode, no changes applied).\n", originalWordStr)
						} else {
							fmt.Fprintf(os.Stderr, "WARNING (Manual Mode): Term '%s' not found in dictionary. Removing ':rubi' suffix.\n", originalWordStr)
						}
					}
				}
			}
			return ast.WalkContinue, nil
		default:
			return ast.WalkContinue, nil
		}
	}

	if err := ast.Walk(document, walker); err != nil {
		return nil, fmt.Errorf("error during AST traversal: %w", err)
	}

	if dryRun || len(patches) == 0 {
		return content, nil
	}

	return ApplyPatches(content, patches)
}