package render

import (
	"strings"
	"unicode/utf8"
)

type Lexer struct {
	input         string
	pos           int  // current position in input (points to current char)
	readPos       int  // current reading position (after current char)
	ch            rune // current char
	line          int
	col           int
	bufferedToken *Token // a token buffered to be returned on the next call
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, col: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		r, size := utf8.DecodeRuneInString(l.input[l.readPos:])
		l.ch = r
		l.pos = l.readPos
		l.readPos += size

		if l.ch == '\n' {
			l.line++
			l.col = 0
		} else {
			l.col++
		}
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return r
}

func (l *Lexer) NextToken() Token {
	var tok Token
	tok.Line = l.line
	tok.Col = l.col

	// If token was buffered, return it now
	if l.bufferedToken != nil {
		t := *l.bufferedToken
		l.bufferedToken = nil
		return t
	}

	skippedSpace := l.skipWhitespaceTracked()

	switch l.ch {
	case '*':
		if l.peekChar() == '*' {
			l.readChar()
			tok = l.newToken(TokenDoubleStar, "**")
		} else {
			tok = l.newToken(TokenStar, "*")
		}
	case '[':
		if l.peekChar() == '[' {
			l.readChar()
			tok = l.newToken(TokenDoubleLBracket, "[[")
		} else {
			tok = l.newToken(TokenLBracket, "[")
		}
	case ']':
		if l.peekChar() == ']' {
			l.readChar()
			tok = l.newToken(TokenDoubleRBracket, "]]")
		} else {
			tok = l.newToken(TokenRBracket, "]")
		}
	case '(':
		tok = l.newToken(TokenLParen, "(")
	case ')':
		tok = l.newToken(TokenRParen, ")")
	case '`':
		if l.peekChar() == '`' && l.input[l.readPos:l.readPos+2] == "``" {
			// triple backtick
			if l.readPos+2 < len(l.input) && l.input[l.readPos+2] == '`' {
				l.readChar()
				l.readChar()
				tok = l.newToken(TokenTripleBacktick, "```")
			} else {
				tok = l.newToken(TokenBacktick, "`")
			}
		} else {
			tok = l.newToken(TokenBacktick, "`")
		}
	case '#':
		tok = l.newToken(TokenHash, "#")
	case '-':
		tok = l.newToken(TokenDash, "-")
	case '\n':
		tok = l.newToken(TokenNewline, "\n")
	case 0:
		tok.Literal = ""
		tok.Type = TokenEOF
	default:
		if isTextChar(l.ch) {
			literal := l.readText()
			if skippedSpace && tok.Col > 1 {
				literal = " " + literal
			}
			tok = Token{Type: TokenText, Literal: literal, Line: tok.Line, Col: tok.Col}
			return tok // already advanced
		} else {
			tok = l.newToken(TokenText, string(l.ch))
		}
	}

	l.readChar()

	// If whitespace was skipped mid-line before a non-text token,
	// return the space token first and buffer the real token for next call
	if skippedSpace && tok.Col > 1 && tok.Type != TokenNewline && tok.Type != TokenEOF {
		l.bufferedToken = &tok
		return Token{Type: TokenText, Literal: " ", Line: tok.Line, Col: tok.Col}
	}

	return tok
}

func (l *Lexer) newToken(tokenType TokenType, literal string) Token {
	return Token{Type: tokenType, Literal: literal, Line: l.line, Col: l.col}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// skipWhitespaceTracked skips whitespace, true if any consumed.
func (l *Lexer) skipWhitespaceTracked() bool {
	skipped := false
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		skipped = true
		l.readChar()
	}
	return skipped
}

func (l *Lexer) readText() string {
	position := l.pos
	for isTextChar(l.ch) && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.pos]
}

func isTextChar(ch rune) bool {
	return ch != '*' && ch != '[' && ch != ']' && ch != '(' && ch != ')' &&
		ch != '`' && ch != '#' && ch != '\n' && ch != 0 && ch != '-'
}

// ReadCodeBlock read everything until closing ```
func (l *Lexer) ReadCodeBlock() string {
	l.readChar() // skip initial `
	l.readChar() // skip second `
	l.readChar() // skip third `

	var sb strings.Builder
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '`' && l.peekChar() == '`' && l.readPos+1 < len(l.input) && l.input[l.readPos+1] == '`' {
			l.readChar()
			l.readChar()
			l.readChar()
			break
		}
		sb.WriteRune(l.ch)
		l.readChar()
	}
	return sb.String()
}

// ReadInlineCode reads until closing `
func (l *Lexer) ReadInlineCode() string {
	l.readChar() // skip initial `
	var sb strings.Builder
	for {
		if l.ch == 0 || l.ch == '\n' {
			break
		}
		if l.ch == '`' {
			l.readChar()
			break
		}
		sb.WriteRune(l.ch)
		l.readChar()
	}
	return sb.String()
}
