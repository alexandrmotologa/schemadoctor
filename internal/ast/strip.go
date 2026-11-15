package ast

import (
	"bytes"
)

// StripMarkdownAndChatter extracts the core JSON substring from an LLM response.
// It removes leading markdown fences (e.g. ```json), conversational preambles,
// and trailing markdown closers or conversational commentary.
func StripMarkdownAndChatter(input []byte) []byte {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return trimmed
	}

	// Step 1: Find the first candidate JSON root delimiter '{' or '['.
	firstBrace := bytes.IndexByte(trimmed, '{')
	firstBracket := bytes.IndexByte(trimmed, '[')

	startIdx := -1
	switch {
	case firstBrace != -1 && firstBracket != -1:
		if firstBrace < firstBracket {
			startIdx = firstBrace
		} else {
			startIdx = firstBracket
		}
	case firstBrace != -1:
		startIdx = firstBrace
	case firstBracket != -1:
		startIdx = firstBracket
	default:
		// No bracket or brace found. Return trimmed as-is.
		return trimmed
	}

	trimmed = trimmed[startIdx:]

	// Step 2: Strip trailing markdown fences (```) or closing code block tags if present.
	// Look from the end backwards.
	if fenceIdx := bytes.LastIndex(trimmed, []byte("```")); fenceIdx != -1 {
		// Only strip the fence if it comes after some JSON content
		trimmed = bytes.TrimSpace(trimmed[:fenceIdx])
	}

	return trimmed
}
