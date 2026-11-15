package ast

import (
	"bytes"
	"unicode"
)

// Lexer converts a byte slice into a sequence of JSON tokens.
type Lexer struct {
	src    []byte
	pos    int
	length int
}

// NewLexer creates a Lexer targeting the provided source slice.
func NewLexer(src []byte) *Lexer {
	return &Lexer{
		src:    src,
		pos:    0,
		length: len(src),
	}
}

// Next returns the next token from the input.
func (l *Lexer) Next() Token {
	l.skipWhitespace()

	if l.pos >= l.length {
		return Token{Type: TokenEOF, Offset: l.pos}
	}

	start := l.pos
	ch := l.src[l.pos]

	switch ch {
	case '{':
		l.pos++
		return Token{Type: TokenOpenBrace, Offset: start, Length: 1, Raw: l.src[start:l.pos]}
	case '}':
		l.pos++
		return Token{Type: TokenCloseBrace, Offset: start, Length: 1, Raw: l.src[start:l.pos]}
	case '[':
		l.pos++
		return Token{Type: TokenOpenBracket, Offset: start, Length: 1, Raw: l.src[start:l.pos]}
	case ']':
		l.pos++
		return Token{Type: TokenCloseBracket, Offset: start, Length: 1, Raw: l.src[start:l.pos]}
	case ':':
		l.pos++
		return Token{Type: TokenColon, Offset: start, Length: 1, Raw: l.src[start:l.pos]}
	case ',':
		l.pos++
		return Token{Type: TokenComma, Offset: start, Length: 1, Raw: l.src[start:l.pos]}
	case '"':
		return l.scanDoubleQuoteString()
	case '\'':
		return l.scanSingleQuoteString()
	default:
		if ch == '-' || (ch >= '0' && ch <= '9') {
			return l.scanNumber()
		}
		if isIdentStart(ch) {
			return l.scanIdentifier()
		}
		l.pos++
		return Token{Type: TokenUnknown, Offset: start, Length: 1, Raw: l.src[start:l.pos]}
	}
}

// Pos returns the current reading index.
func (l *Lexer) Pos() int {
	return l.pos
}

// SetPos moves the reading index.
func (l *Lexer) SetPos(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > l.length {
		pos = l.length
	}
	l.pos = pos
}

func (l *Lexer) skipWhitespace() {
	for l.pos < l.length {
		c := l.src[l.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			l.pos++
		} else {
			break
		}
	}
}

func (l *Lexer) scanDoubleQuoteString() Token {
	start := l.pos
	l.pos++ // consume opening quote

	for l.pos < l.length {
		c := l.src[l.pos]
		if c == '\\' {
			l.pos += 2 // skip escaped character
			continue
		}
		if c == '"' {
			// Check if this quote is followed by structural tokens (, : } ])
			// or if it might be an unescaped quote inside string text.
			if l.isPlausibleStringTerminator(l.pos + 1) {
				l.pos++ // consume closing quote
				return Token{Type: TokenString, Offset: start, Length: l.pos - start, Raw: l.src[start:l.pos]}
			}
			// If not plausible string terminator and followed by valid string characters,
			// continue scanning as inner content.
			l.pos++
			continue
		}
		l.pos++
	}

	// Unterminated string (reached EOF before closing quote)
	return Token{Type: TokenString, Offset: start, Length: l.pos - start, Raw: l.src[start:l.pos]}
}

// isPlausibleStringTerminator checks if the character after a quote indicates valid JSON structure.
func (l *Lexer) isPlausibleStringTerminator(pos int) bool {
	for pos < l.length {
		c := l.src[pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			pos++
			continue
		}
		// Valid characters following a JSON string:
		// : (after key)
		// , (after value)
		// } (closing object)
		// ] (closing array)
		return c == ':' || c == ',' || c == '}' || c == ']'
	}
	// EOF directly after quote is also a valid terminator
	return true
}

func (l *Lexer) scanSingleQuoteString() Token {
	start := l.pos
	l.pos++ // consume opening single quote

	for l.pos < l.length {
		c := l.src[l.pos]
		if c == '\\' {
			l.pos += 2
			continue
		}
		if c == '\'' {
			l.pos++
			return Token{Type: TokenString, Offset: start, Length: l.pos - start, Raw: l.src[start:l.pos]}
		}
		l.pos++
	}

	return Token{Type: TokenString, Offset: start, Length: l.pos - start, Raw: l.src[start:l.pos]}
}

func (l *Lexer) scanNumber() Token {
	start := l.pos
	if l.src[l.pos] == '-' {
		l.pos++
	}

	for l.pos < l.length && (l.src[l.pos] >= '0' && l.src[l.pos] <= '9') {
		l.pos++
	}

	if l.pos < l.length && l.src[l.pos] == '.' {
		l.pos++
		for l.pos < l.length && (l.src[l.pos] >= '0' && l.src[l.pos] <= '9') {
			l.pos++
		}
	}

	if l.pos < l.length && (l.src[l.pos] == 'e' || l.src[l.pos] == 'E') {
		l.pos++
		if l.pos < l.length && (l.src[l.pos] == '+' || l.src[l.pos] == '-') {
			l.pos++
		}
		for l.pos < l.length && (l.src[l.pos] >= '0' && l.src[l.pos] <= '9') {
			l.pos++
		}
	}

	return Token{Type: TokenNumber, Offset: start, Length: l.pos - start, Raw: l.src[start:l.pos]}
}

func (l *Lexer) scanIdentifier() Token {
	start := l.pos
	for l.pos < l.length && (isIdentPart(l.src[l.pos])) {
		l.pos++
	}

	raw := l.src[start:l.pos]
	lower := bytes.ToLower(raw)

	if bytes.Equal(lower, []byte("true")) {
		return Token{Type: TokenTrue, Offset: start, Length: len(raw), Raw: raw}
	}
	if bytes.Equal(lower, []byte("false")) {
		return Token{Type: TokenFalse, Offset: start, Length: len(raw), Raw: raw}
	}
	if bytes.Equal(lower, []byte("null")) || bytes.Equal(lower, []byte("none")) {
		return Token{Type: TokenNull, Offset: start, Length: len(raw), Raw: raw}
	}

	return Token{Type: TokenIdentifier, Offset: start, Length: len(raw), Raw: raw}
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9') || c == '-' || unicode.IsLetter(rune(c))
}
