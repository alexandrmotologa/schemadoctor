package ast

// TokenType identifies a lexical token in a JSON character stream.
type TokenType int

const (
	TokenUnknown TokenType = iota
	TokenEOF
	TokenOpenBrace     // {
	TokenCloseBrace    // }
	TokenOpenBracket   // [
	TokenCloseBracket  // ]
	TokenColon         // :
	TokenComma         // ,
	TokenString        // "..."
	TokenNumber        // 123, -4.56, 1e10
	TokenTrue          // true
	TokenFalse         // false
	TokenNull          // null
	TokenIdentifier    // unquoted identifier (e.g. unquoted keys or True/False/None)
)

// Token represents a token slice within the source buffer.
type Token struct {
	Type   TokenType
	Offset int
	Length int
	Raw    []byte
}
