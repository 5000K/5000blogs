package render

import "fmt"

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

const (
	TokenEOF     TokenType = "EOF"
	TokenNewline TokenType = "NEWLINE"
	TokenText    TokenType = "TEXT"

	TokenStar           TokenType = "STAR"            // *
	TokenDoubleStar     TokenType = "DOUBLE_STAR"     // **
	TokenBacktick       TokenType = "BACKTICK"        // `
	TokenTripleBacktick TokenType = "TRIPLE_BACKTICK" // ```

	TokenLBracket       TokenType = "LBRACKET"        // [
	TokenRBracket       TokenType = "RBRACKET"        // ]
	TokenLParen         TokenType = "LPAREN"          // (
	TokenRParen         TokenType = "RPAREN"          // )
	TokenDoubleLBracket TokenType = "DOUBLE_LBRACKET" // [[
	TokenDoubleRBracket TokenType = "DOUBLE_RBRACKET" // ]]

	TokenHash TokenType = "HASH" // #

	TokenDash TokenType = "DASH" // -
)

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %d:%d)", t.Type, t.Literal, t.Line, t.Col)
}
