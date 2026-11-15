package ast

import (
	"bytes"
	"encoding/json"
	"strings"
)

// RepairFlags tracks which repair operations were applied during healing.
type RepairFlags struct {
	StrippedMarkdown    bool `json:"stripped_markdown"`
	ClosedOpenString    bool `json:"closed_open_string"`
	BalancedDelimiters  bool `json:"balanced_delimiters"`
	RemovedTrailingComma bool `json:"removed_trailing_comma"`
	CompletedDanglingKey bool `json:"completed_dangling_key"`
	NormalizedIdent     bool `json:"normalized_ident"`
	FixedUnescapedQuote bool `json:"fixed_unescaped_quote"`
}

// Any returns true if any repair flag was triggered.
func (f RepairFlags) Any() bool {
	return f.StrippedMarkdown || f.ClosedOpenString || f.BalancedDelimiters ||
		f.RemovedTrailingComma || f.CompletedDanglingKey || f.NormalizedIdent || f.FixedUnescapedQuote
}

// Summary returns a comma-separated list of active repairs.
func (f RepairFlags) Summary() string {
	var active []string
	if f.StrippedMarkdown {
		active = append(active, "stripped_markdown")
	}
	if f.ClosedOpenString {
		active = append(active, "closed_open_string")
	}
	if f.BalancedDelimiters {
		active = append(active, "balanced_delimiters")
	}
	if f.RemovedTrailingComma {
		active = append(active, "removed_trailing_comma")
	}
	if f.CompletedDanglingKey {
		active = append(active, "completed_dangling_key")
	}
	if f.NormalizedIdent {
		active = append(active, "normalized_ident")
	}
	if f.FixedUnescapedQuote {
		active = append(active, "fixed_unescaped_quote")
	}
	return strings.Join(active, ", ")
}

// RepairResult encapsulates the healed output and its diagnostic flags.
type RepairResult struct {
	Original []byte
	Repaired []byte
	Flags    RepairFlags
	Valid    bool
}

// Repair takes potentially malformed JSON bytes from an LLM and deterministically heals syntax errors.
func Repair(input []byte) RepairResult {
	res := RepairResult{
		Original: input,
		Flags:    RepairFlags{},
	}

	// If input is already strictly valid JSON, return immediately
	if json.Valid(input) {
		res.Repaired = input
		res.Valid = true
		return res
	}

	// 1. Strip markdown code fences (```json) and conversational chatter
	stripped := StripMarkdownAndChatter(input)
	if len(stripped) != len(input) {
		res.Flags.StrippedMarkdown = true
	}

	// Quick check if stripped version is now valid JSON
	if json.Valid(stripped) {
		res.Repaired = stripped
		res.Valid = true
		return res
	}

	// 2. Tokenize and reconstruct with stack tracking
	healed, flags := reconstruct(stripped)
	res.Flags.ClosedOpenString = res.Flags.ClosedOpenString || flags.ClosedOpenString
	res.Flags.BalancedDelimiters = res.Flags.BalancedDelimiters || flags.BalancedDelimiters
	res.Flags.RemovedTrailingComma = res.Flags.RemovedTrailingComma || flags.RemovedTrailingComma
	res.Flags.CompletedDanglingKey = res.Flags.CompletedDanglingKey || flags.CompletedDanglingKey
	res.Flags.NormalizedIdent = res.Flags.NormalizedIdent || flags.NormalizedIdent
	res.Flags.FixedUnescapedQuote = res.Flags.FixedUnescapedQuote || flags.FixedUnescapedQuote

	// 3. Final validation check
	res.Repaired = healed
	res.Valid = json.Valid(healed)

	return res
}

func reconstruct(src []byte) ([]byte, RepairFlags) {
	flags := RepairFlags{}
	var out bytes.Buffer
	out.Grow(len(src) + 64)

	stack := NewStack(32)
	lexer := NewLexer(src)

	// State tracking:
	// lastMeaningfulToken: the token right before current
	var lastType TokenType = TokenUnknown
	expectingValue := false // after colon or at start of element

	for {
		tok := lexer.Next()
		if tok.Type == TokenEOF {
			break
		}

		switch tok.Type {
		case TokenOpenBrace:
			out.WriteByte('{')
			stack.Push(DelimObject)
			lastType = TokenOpenBrace
			expectingValue = false

		case TokenCloseBrace:
			// If preceding token was a comma, trailing comma was written to out; strip it
			if lastType == TokenComma {
				trimTrailingComma(&out)
				flags.RemovedTrailingComma = true
			} else if lastType == TokenColon {
				out.WriteString("null")
				flags.CompletedDanglingKey = true
			}
			out.WriteByte('}')
			stack.Pop()
			lastType = TokenCloseBrace
			expectingValue = false

		case TokenOpenBracket:
			out.WriteByte('[')
			stack.Push(DelimArray)
			lastType = TokenOpenBracket
			expectingValue = true

		case TokenCloseBracket:
			if lastType == TokenComma {
				trimTrailingComma(&out)
				flags.RemovedTrailingComma = true
			} else if lastType == TokenColon {
				out.WriteString("null")
				flags.CompletedDanglingKey = true
			}
			out.WriteByte(']')
			stack.Pop()
			lastType = TokenCloseBracket
			expectingValue = false

		case TokenColon:
			// If preceding token was a comma or colon, handle gracefully
			out.WriteByte(':')
			lastType = TokenColon
			expectingValue = true

		case TokenComma:
			// Avoid double commas (,,)
			if lastType == TokenComma || lastType == TokenOpenBrace || lastType == TokenOpenBracket {
				// skip duplicate or invalid comma
				flags.RemovedTrailingComma = true
				continue
			}
			if lastType == TokenColon {
				// Dangling key: "foo":,
				out.WriteString("null")
				flags.CompletedDanglingKey = true
			}
			out.WriteByte(',')
			lastType = TokenComma
			expectingValue = false

		case TokenString:
			formattedStr, closed, fixedQuote := normalizeString(tok.Raw)
			if closed {
				flags.ClosedOpenString = true
			}
			if fixedQuote {
				flags.FixedUnescapedQuote = true
			}
			out.Write(formattedStr)
			lastType = TokenString
			expectingValue = false

		case TokenNumber:
			out.Write(tok.Raw)
			lastType = tok.Type
			expectingValue = false

		case TokenTrue:
			if !bytes.Equal(tok.Raw, []byte("true")) {
				flags.NormalizedIdent = true
			}
			out.WriteString("true")
			lastType = TokenTrue
			expectingValue = false

		case TokenFalse:
			if !bytes.Equal(tok.Raw, []byte("false")) {
				flags.NormalizedIdent = true
			}
			out.WriteString("false")
			lastType = TokenFalse
			expectingValue = false

		case TokenNull:
			if !bytes.Equal(tok.Raw, []byte("null")) {
				flags.NormalizedIdent = true
			}
			out.WriteString("null")
			lastType = TokenNull
			expectingValue = false

		case TokenIdentifier:
			// Unquoted identifier or Python constant
			lower := bytes.ToLower(tok.Raw)
			switch {
			case bytes.Equal(lower, []byte("true")):
				out.WriteString("true")
				flags.NormalizedIdent = true
				lastType = TokenTrue
			case bytes.Equal(lower, []byte("false")):
				out.WriteString("false")
				flags.NormalizedIdent = true
				lastType = TokenFalse
			case bytes.Equal(lower, []byte("none")) || bytes.Equal(lower, []byte("null")):
				out.WriteString("null")
				flags.NormalizedIdent = true
				lastType = TokenNull
			default:
				// If expecting a key or at root object, quote it as a string
				top, inScope := stack.Peek()
				if inScope && top == DelimObject && !expectingValue {
					out.WriteByte('"')
					out.Write(tok.Raw)
					out.WriteByte('"')
					flags.NormalizedIdent = true
					lastType = TokenString
				} else {
					// Coerce unknown ident as string value
					out.WriteByte('"')
					out.Write(tok.Raw)
					out.WriteByte('"')
					flags.NormalizedIdent = true
					lastType = TokenString
				}
			}
			expectingValue = false

		default:
			// Unknown byte, skip or write if safe
		}
	}

	// Stream cut off or unexpected EOF!
	// 1. If last token was a trailing comma, remove it
	if lastType == TokenComma {
		trimTrailingComma(&out)
		flags.RemovedTrailingComma = true
	}

	// 2. If last token was a colon (dangling key), inject null
	if lastType == TokenColon {
		out.WriteString("null")
		flags.CompletedDanglingKey = true
	}

	// 3. Close open delimiters in reverse stack order
	if stack.Len() > 0 {
		flags.BalancedDelimiters = true
		for {
			delim, ok := stack.Pop()
			if !ok {
				break
			}
			if delim == DelimObject {
				out.WriteByte('}')
			} else if delim == DelimArray {
				out.WriteByte(']')
			}
		}
	}

	return out.Bytes(), flags
}

// CalculateClosingSuffix determines the exact characters required to close an incomplete JSON stream at EOF.
func CalculateClosingSuffix(src []byte) []byte {
	stripped := StripMarkdownAndChatter(src)
	if len(stripped) == 0 {
		return nil
	}

	stack := NewStack(32)
	lexer := NewLexer(stripped)

	var lastToken Token
	var expectingColon = false
	var expectingValue = false

	for {
		tok := lexer.Next()
		if tok.Type == TokenEOF {
			break
		}

		switch tok.Type {
		case TokenOpenBrace:
			stack.Push(DelimObject)
			expectingValue = false
			expectingColon = false
		case TokenCloseBrace:
			stack.Pop()
			expectingValue = false
			expectingColon = false
		case TokenOpenBracket:
			stack.Push(DelimArray)
			expectingValue = true
			expectingColon = false
		case TokenCloseBracket:
			stack.Pop()
			expectingValue = false
			expectingColon = false
		case TokenColon:
			expectingValue = true
			expectingColon = false
		case TokenComma:
			expectingValue = false
			expectingColon = false
		case TokenString:
			if expectingValue {
				expectingValue = false
			} else {
				expectingColon = true
			}
		default:
			expectingValue = false
		}

		lastToken = tok
	}

	var suffix bytes.Buffer

	// If last token was an unclosed string
	if lastToken.Type == TokenString && len(lastToken.Raw) > 0 {
		c := lastToken.Raw[len(lastToken.Raw)-1]
		if c != '"' && c != '\'' {
			suffix.WriteByte('"')
		}
	}

	if expectingColon {
		suffix.WriteString(": null")
	} else if expectingValue {
		suffix.WriteString("null")
	}

	// Close open delimiters in reverse order
	for stack.Len() > 0 {
		delim, ok := stack.Pop()
		if !ok {
			break
		}
		if delim == DelimObject {
			suffix.WriteByte('}')
		} else if delim == DelimArray {
			suffix.WriteByte(']')
		}
	}

	return suffix.Bytes()
}

func trimTrailingComma(buf *bytes.Buffer) {
	b := buf.Bytes()
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == ',' {
			buf.Truncate(i)
			return
		}
		if b[i] != ' ' && b[i] != '\t' && b[i] != '\r' && b[i] != '\n' {
			return
		}
	}
}

// normalizeString handles single quote conversion, escaping inner unescaped quotes,
// and closing unterminated strings.
func normalizeString(raw []byte) ([]byte, bool, bool) {
	if len(raw) == 0 {
		return []byte(`""`), false, false
	}

	closedString := false
	fixedQuote := false

	// Check if single-quoted string
	if raw[0] == '\'' {
		content := raw[1:]
		if len(content) > 0 && content[len(content)-1] == '\'' {
			content = content[:len(content)-1]
		} else {
			closedString = true
		}
		// Escape any internal unescaped double quotes and replace \' with '
		escaped := escapeDoubleQuotes(content)
		var buf bytes.Buffer
		buf.WriteByte('"')
		buf.Write(escaped)
		buf.WriteByte('"')
		return buf.Bytes(), closedString, true
	}

	// Double-quoted string
	if raw[0] == '"' {
		content := raw[1:]
		if len(content) > 0 && content[len(content)-1] == '"' {
			content = content[:len(content)-1]
		} else {
			closedString = true
		}

		// Escape any internal unescaped double quotes
		escaped := escapeDoubleQuotes(content)
		if len(escaped) != len(content) {
			fixedQuote = true
		}

		var buf bytes.Buffer
		buf.WriteByte('"')
		buf.Write(escaped)
		buf.WriteByte('"')
		return buf.Bytes(), closedString, fixedQuote
	}

	// Raw content without leading quote
	var buf bytes.Buffer
	buf.WriteByte('"')
	buf.Write(escapeDoubleQuotes(raw))
	buf.WriteByte('"')
	return buf.Bytes(), true, true
}

func escapeDoubleQuotes(s []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(s) + 8)

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' {
			buf.WriteByte(c)
			if i+1 < len(s) {
				i++
				buf.WriteByte(s[i])
			}
			continue
		}
		if c == '"' {
			buf.WriteString(`\"`)
			continue
		}
		buf.WriteByte(c)
	}
	return buf.Bytes()
}
