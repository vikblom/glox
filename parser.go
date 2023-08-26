package glox

import "fmt"

type Parser struct {
	tokens  []Token
	current int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
	}
}

func (p *Parser) parseExpr() Expr {
	return p.parseEquality()
}

func (p *Parser) parseEquality() Expr {
	expr := p.parseComparison()
	for p.match(BANG_EQUAL, EQUAL_EQUAL) {
		expr = &BinaryExpr{
			op:    p.previous(),
			left:  expr,
			right: p.parseComparison(),
		}
	}
	return expr
}

func (p *Parser) parseComparison() Expr {
	expr := p.parseTerm()
	for p.match(GREATER, GREATER_EQUAL, LESS, LESS_EQUAL) {
		expr = &BinaryExpr{
			op:    p.previous(),
			left:  expr,
			right: p.parseTerm(),
		}
	}
	return expr
}

func (p *Parser) parseTerm() Expr {
	expr := p.parseFactor()
	for p.match(PLUS, DASH) {
		expr = &BinaryExpr{
			op:    p.previous(),
			left:  expr,
			right: p.parseFactor(),
		}
	}
	return expr
}

func (p *Parser) parseFactor() Expr {
	expr := p.parseUnary()
	for p.match(STAR, SLASH) {
		expr = &BinaryExpr{
			op:    p.previous(),
			left:  expr,
			right: p.parseUnary(),
		}
	}
	return expr
}

func (p *Parser) parseUnary() Expr {
	if p.match(DASH, BANG) {
		return &UnaryExpr{
			op:    p.previous(),
			right: p.parseUnary(),
		}
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() Expr {

	switch {
	case p.match(FALSE):
		return &Literal{val: false}
	case p.match(TRUE):
		return &Literal{val: true}
	case p.match(NIL):
		return &Literal{val: nil}
	case p.match(NUMBER, STRING):
		return &Literal{val: p.previous().Literal}
	case p.match(PAREN_LEFT):
		expr := p.parseExpr()
		p.consume(PAREN_RIGHT, "Expected closing ')'")
		return &Grouping{group: expr}
	}

	return nil
}

// match if currently on one of token types.
// Behaves like a "consume", moving forward if matching.
func (p *Parser) match(tts ...TokenType) bool {
	at := p.tokens[p.current]
	if at.Kind == EOF {
		return false
	}
	for _, t := range tts {
		if t == at.Kind {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) consume(tt TokenType, msg string) {
	at := p.peek()
	if at.Kind != tt {
		p.error(at.Line, msg)
		p.sync()
		return
	}

	p.advance()
}

func (p *Parser) error(line int, msg string) {
	// Don't try to emulate exceptions with panic.
	// Just report and try to sync at the bottom of the callstack like
	// the Go parser does.
	fmt.Printf("error on line %d: %s", line, msg)
}

func (p *Parser) sync() {
	p.advance()
	for !p.isAtEnd() {
		if p.previous().Kind == SEMICOLON {
			return
		}
		switch p.peek().Kind {
		case CLASS, FOR, FUN, IF, PRINT, RETURN, VAR, WHILE:
			return
		}
		p.advance()
	}
}

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.current += 1
	}
	return p.previous()
}

func (p *Parser) previous() Token {
	return p.tokens[p.current-1]
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Kind == EOF
}

func (p *Parser) peek() Token {
	return p.tokens[p.current]
}
