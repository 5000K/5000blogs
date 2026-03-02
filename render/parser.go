package render

import (
	"fmt"
	"strings"
)

type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token
	context   []*Node // Stack for current nesting context
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l}
	// set curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Parse() (*Node, error) {
	doc := &Node{Type: NodeDocument, Attributes: make(map[string]string)}
	p.context = []*Node{doc}

	// Handle frontmatter if present
	if p.curToken.Type == TokenDash && p.peekToken.Type == TokenDash {
		if err := p.parseFrontmatter(); err != nil {
			return nil, err
		}
	}

	for p.curToken.Type != TokenEOF {
		if err := p.parseBlock(); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.curToken.Line, err)
		}
	}

	return doc, nil
}

func (p *Parser) currentContext() *Node {
	if len(p.context) == 0 {
		return nil
	}
	return p.context[len(p.context)-1]
}

func (p *Parser) pushContext(n *Node) {
	p.context = append(p.context, n)
}

func (p *Parser) popContext() {
	if len(p.context) > 0 {
		p.context = p.context[:len(p.context)-1]
	}
}

func (p *Parser) addToCurrent(n *Node) error {
	ctx := p.currentContext()
	if ctx == nil {
		return fmt.Errorf("no context to add to")
	}
	return ctx.AddChild(n)
}

func (p *Parser) parseFrontmatter() error {
	// Consume ---
	p.nextToken()
	p.nextToken()

	var lines []string
	for p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenDash && p.peekToken.Type == TokenDash {

			p.nextToken()
			p.nextToken()
			if p.curToken.Type == TokenDash {
				p.nextToken() // consume closing ---
				break
			}
		}
		if p.curToken.Type == TokenNewline {
			lines = append(lines, "")
		} else {
			lines = append(lines, p.curToken.Literal)
		}
		p.nextToken()
	}

	fmNode := &Node{
		Type:    NodeFrontmatter,
		Literal: strings.Join(lines, "\n"),
	}
	return p.addToCurrent(fmNode)
}

func (p *Parser) parseBlock() error {
	switch p.curToken.Type {
	case TokenNewline:
		p.nextToken()
		return nil
	case TokenHash:
		return p.parseHeader()
	case TokenTripleBacktick:
		return p.parseCodeBlock()
	default:
		return p.parseParagraph()
	}
}

func (p *Parser) parseHeader() error {
	// Count hashes
	level := 0
	for p.curToken.Type == TokenHash && level < 6 {
		level++
		p.nextToken()
	}

	// Skip space after hashes
	if p.curToken.Type == TokenText && strings.HasPrefix(p.curToken.Literal, " ") {
		p.curToken.Literal = strings.TrimLeft(p.curToken.Literal, " ")
		if p.curToken.Literal == "" {
			p.nextToken()
		}
	}

	header := &Node{Type: NodeHeader, Level: level}
	p.pushContext(header)

	// Parse inline until newline
	for p.curToken.Type != TokenNewline && p.curToken.Type != TokenEOF {
		if err := p.parseInline(); err != nil {
			return err
		}
	}

	p.popContext()
	if err := p.addToCurrent(header); err != nil {
		return err
	}
	p.nextToken() // consume newline
	return nil
}

func (p *Parser) parseCodeBlock() error {
	p.nextToken() // skip opening ```

	// Check for language identifier
	lang := ""
	if p.curToken.Type == TokenText {
		lang = p.curToken.Literal
		p.nextToken()
	}

	var content strings.Builder
	for p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenTripleBacktick {
			p.nextToken()
			break
		}
		content.WriteString(p.curToken.Literal)
		if p.curToken.Type == TokenNewline {
			content.WriteString("\n")
		}
		p.nextToken()
	}

	codeBlock := &Node{
		Type:       NodeCodeBlock,
		Literal:    content.String(),
		Attributes: map[string]string{"lang": lang},
	}
	return p.addToCurrent(codeBlock)
}

func (p *Parser) parseParagraph() error {
	para := &Node{Type: NodeParagraph}
	p.pushContext(para)

	for p.curToken.Type != TokenEOF {
		// Paragraph ends on double newline / block-level token
		if p.curToken.Type == TokenNewline && p.peekToken.Type == TokenNewline {
			p.nextToken() // consume first newline
			break
		}
		if p.isBlockStart() {
			break
		}

		if err := p.parseInline(); err != nil {
			return err
		}
	}

	p.popContext()

	// Don't add empty paragraphs
	if len(para.Children) > 0 {
		if err := p.addToCurrent(para); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) isBlockStart() bool {
	switch p.curToken.Type {
	case TokenHash, TokenTripleBacktick, TokenDash:
		return true
	default:
		return false
	}
}

func (p *Parser) parseInline() error {
	switch p.curToken.Type {
	case TokenDoubleStar:
		return p.parseBold()
	case TokenStar:
		return p.parseItalic()
	case TokenLBracket:
		return p.parseLink()
	case TokenDoubleLBracket:
		return p.parseWikiLink()
	case TokenBacktick:
		return p.parseInlineCode()
	case TokenNewline:
		br := &Node{Type: NodeSoftBreak}
		if err := p.addToCurrent(br); err != nil {
			return err
		}
		p.nextToken()
		return nil
	case TokenText:
		text := &Node{Type: NodeText, Literal: p.curToken.Literal}
		if err := p.addToCurrent(text); err != nil {
			return err
		}
		p.nextToken()
		return nil
	default:
		// Treat unexpected tokens as text
		text := &Node{Type: NodeText, Literal: p.curToken.Literal}
		if err := p.addToCurrent(text); err != nil {
			return err
		}
		p.nextToken()
		return nil
	}
}

func (p *Parser) parseBold() error {
	p.nextToken() // skip **
	bold := &Node{Type: NodeBold}
	p.pushContext(bold)

	for p.curToken.Type != TokenEOF && p.curToken.Type != TokenDoubleStar {
		if err := p.parseInline(); err != nil {
			return err
		}
	}

	p.popContext()
	if err := p.addToCurrent(bold); err != nil {
		return err
	}
	if p.curToken.Type == TokenDoubleStar {
		p.nextToken()
	}
	return nil
}

func (p *Parser) parseItalic() error {
	p.nextToken() // skip *
	italic := &Node{Type: NodeItalic}
	p.pushContext(italic)

	for p.curToken.Type != TokenEOF && p.curToken.Type != TokenStar {
		if err := p.parseInline(); err != nil {
			return err
		}
	}

	p.popContext()
	if err := p.addToCurrent(italic); err != nil {
		return err
	}
	if p.curToken.Type == TokenStar {
		p.nextToken()
	}
	return nil
}

func (p *Parser) parseLink() error {
	p.nextToken() // skip [

	link := &Node{Type: NodeLink}
	p.pushContext(link)

	// Parse link text
	for p.curToken.Type != TokenEOF && p.curToken.Type != TokenRBracket {
		if err := p.parseInline(); err != nil {
			return err
		}
	}
	p.popContext()

	if p.curToken.Type != TokenRBracket {
		return fmt.Errorf("expected ]")
	}
	p.nextToken()

	// Parse URL
	if p.curToken.Type != TokenLParen {
		return fmt.Errorf("expected ( after ]")
	}
	p.nextToken()

	url := ""
	for p.curToken.Type != TokenEOF && p.curToken.Type != TokenRParen {
		url += p.curToken.Literal
		p.nextToken()
	}
	link.Destination = url

	if err := p.addToCurrent(link); err != nil {
		return err
	}
	p.nextToken() // skip )
	return nil
}

func (p *Parser) parseWikiLink() error {
	p.nextToken() // skip [[

	title := ""
	for p.curToken.Type != TokenEOF && p.curToken.Type != TokenDoubleRBracket {
		title += p.curToken.Literal
		p.nextToken()
	}

	wikiLink := &Node{
		Type:    NodeWikiLink,
		Literal: strings.TrimSpace(title),
	}

	if err := p.addToCurrent(wikiLink); err != nil {
		return err
	}

	if p.curToken.Type == TokenDoubleRBracket {
		p.nextToken()
	}
	return nil
}

func (p *Parser) parseInlineCode() error {
	p.nextToken() // skip `

	code := &Node{Type: NodeCode}
	var content strings.Builder
	for p.curToken.Type != TokenEOF && p.curToken.Type != TokenBacktick {
		content.WriteString(p.curToken.Literal)
		p.nextToken()
	}
	code.Literal = content.String()

	if err := p.addToCurrent(code); err != nil {
		return err
	}
	if p.curToken.Type == TokenBacktick {
		p.nextToken()
	}
	return nil
}
