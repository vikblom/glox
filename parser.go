package glox

import (
	"fmt"
	"strconv"
)

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

type parsingError struct{ error }

// TODO: Should take a token for positioning?
func parseErrf(format string, args ...any) {
	panic(parsingError{error: fmt.Errorf(format, args...)})
}

func (p *Parser) Parse() (stmts []Stmt, err error) {
	// This is the synchronization point.
	// The book does it inside parseDecl
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(parsingError); ok {
				err = re.error
			} else {
				panic(r)
			}
		}
	}()

	for !p.isAtEnd() {
		s := p.parseDecl()
		stmts = append(stmts, s)
	}
	return stmts, nil
}

func (p *Parser) parseDecl() Stmt {
	if p.match(FUN) {
		return p.parseFuncStmt("function")
	}
	if p.match(VAR) {
		return p.parseVarStmt()
	}
	if p.match(CLASS) {
		return p.parseClassStmt()
	}
	return p.parseStmt()
}

func (p *Parser) parseFuncStmt(kind string) Stmt {
	name := p.consume(IDENTIFIER, fmt.Sprintf("Expect %q name.", kind))

	p.consume(PAREN_LEFT, fmt.Sprintf("Expected opening '(' after %s name.", kind))

	params := []Token{}
	if !p.check(PAREN_RIGHT) {
		for {
			if len(params) > 255 {
				parseErrf("Can't have more than 255 parameters.")
			}
			params = append(params, p.consume(IDENTIFIER, "Expect parameter name."))
			if !p.match(COMMA) {
				break
			}
		}
	}
	p.consume(PAREN_RIGHT, fmt.Sprintf("Expected closing ')' after %s params.", kind))

	p.consume(BRACE_LEFT, fmt.Sprintf("Expected '{' before %s body.", kind))
	body := p.parseBlockStmt()

	return &FuncStmt{name: name, params: params, body: []Stmt{body}}
}

func (p *Parser) parseVarStmt() Stmt {
	name := p.consume(IDENTIFIER, "Expected variable name.")

	var init Expr
	if p.match(EQUAL) {
		init = p.parseExpr()
	}

	p.consume(SEMICOLON, "Expected terminating ';' after print value.")
	return &VarStmt{name: name, init: init}
}

func (p *Parser) parseClassStmt() Stmt {
	name := p.consume(IDENTIFIER, "Expected class name.")

	var super *Variable
	if p.match(LESS) {
		p.consume(IDENTIFIER, "Expected superclass name.")
		super = &Variable{name: p.previous()}
	}

	p.consume(BRACE_LEFT, "Expected '{' before class body.")
	methods := []Stmt{}
	for !p.check(BRACE_RIGHT) && !p.isAtEnd() {
		methods = append(methods, p.parseFuncStmt("method"))

	}
	p.consume(BRACE_RIGHT, "Expected '}' afterclass body.")
	return &ClassStmt{name: name, super: super, methods: methods}
}

func (p *Parser) parseStmt() Stmt {
	if p.match(IF) {
		return p.parseIfStmt()
	}
	if p.match(PRINT) {
		return p.parsePrintStmt()
	}
	if p.match(RETURN) {
		return p.parseReturnStmt()
	}
	if p.match(WHILE) {
		return p.parseWhileStmt()
	}
	if p.match(FOR) {
		return p.parseForStmt()
	}
	if p.match(BRACE_LEFT) {
		return p.parseBlockStmt()
	}
	return p.parseExprStmt()
}

func (p *Parser) parseIfStmt() Stmt {
	p.consume(PAREN_LEFT, "Expected opening '(' for if condition.")
	cond := p.parseExpr()
	p.consume(PAREN_RIGHT, "Expected closing ')' for if condition.")

	// then and else statements can be blocks, or not.
	thenBranch := p.parseStmt()

	var elseBranch Stmt
	if p.match(ELSE) {
		elseBranch = p.parseStmt()
	}

	return &IfStmt{cond: cond, thenBranch: thenBranch, elseBranch: elseBranch}
}

func (p *Parser) parsePrintStmt() Stmt {
	val := p.parseExpr()
	p.consume(SEMICOLON, "Expected terminating ';' after print value.")
	return &PrintStmt{expr: val}
}

func (p *Parser) parseReturnStmt() Stmt {
	keyword := p.previous()
	var value Expr
	if !p.check(SEMICOLON) {
		value = p.parseExpr()
	}
	p.consume(SEMICOLON, "Expected terminating ';' after return value.")
	return &ReturnStmt{keyword: keyword, value: value}
}

func (p *Parser) parseWhileStmt() Stmt {
	p.consume(PAREN_LEFT, "Expected opening '(' for while condition.")
	cond := p.parseExpr()
	p.consume(PAREN_RIGHT, "Expected closing ')' for while condition.")
	body := p.parseStmt()
	return &WhileStmt{cond: cond, body: body}
}

func (p *Parser) parseForStmt() Stmt {
	p.consume(PAREN_LEFT, "Expected opening '(' after 'for'.")

	var init Stmt
	switch {
	case p.match(SEMICOLON):
		// Skipped initializer.
	case p.match(VAR):
		init = p.parseVarStmt()
	default:
		init = p.parseExprStmt()
	}

	var cond Expr
	if !p.check(SEMICOLON) {
		cond = p.parseExpr()
	} else {
		cond = &Literal{val: true}
	}
	p.consume(SEMICOLON, "Expected ';' after for loop condition.")

	var incr Expr
	if !p.check(PAREN_RIGHT) {
		incr = p.parseExpr()
	}
	p.consume(PAREN_RIGHT, "Expected ')' after for loop incrementor.")

	body := p.parseStmt()

	// De-sugar into a while loop:
	// {
	//    *init*
	//    while (*cond*) {
	//        *body*
	//        *incr*
	//    }
	// }
	if incr != nil {
		body = &BlockStmt{
			statements: []Stmt{
				body,
				&ExprStmt{expr: incr},
			},
		}
	}

	body = &WhileStmt{cond: cond, body: body}

	if init != nil {
		body = &BlockStmt{
			statements: []Stmt{
				init,
				body,
			},
		}
	}

	return &WhileStmt{cond: cond, body: body}
}

func (p *Parser) parseBlockStmt() Stmt {
	stmts := []Stmt{}
	for !p.check(BRACE_RIGHT) && !p.isAtEnd() {
		stmts = append(stmts, p.parseDecl())
	}
	p.consume(BRACE_RIGHT, "Expected closing '}' after block.")
	return &BlockStmt{statements: stmts}
}

func (p *Parser) parseExprStmt() Stmt {
	val := p.parseExpr()
	p.consume(SEMICOLON, "Expected terminating ';' after expression.")
	return &ExprStmt{expr: val}
}

func (p *Parser) parseExpr() Expr {
	return p.parseAssign()
}

func (p *Parser) parseAssign() Expr {
	expr := p.parseOr()
	if p.match(EQUAL) {
		value := p.parseAssign()
		switch v := expr.(type) {
		case *Variable:
			return &Assign{name: v.name, val: value}
		case *GetExpr:
			return &SetExpr{object: v.object, name: v.name, value: value}
		default:
			runtimeErrf("Invalide assignment target %T", expr)
		}
	}
	return expr
}

func (p *Parser) parseOr() Expr {
	expr := p.parseAnd()
	for p.match(OR) {
		op := p.previous()
		right := p.parseAnd()
		expr = &LogicalExpr{op: op, left: expr, right: right}
	}
	return expr
}

func (p *Parser) parseAnd() Expr {
	expr := p.parseEquality()
	for p.match(AND) {
		op := p.previous()
		right := p.parseEquality()
		expr = &LogicalExpr{op: op, left: expr, right: right}
	}
	return expr
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
	return p.parseCall()
}

func (p *Parser) parseCall() Expr {
	expr := p.parsePrimary()
	for {
		// Looping since we could have repeated calls like: callback()().
		if p.match(PAREN_LEFT) {
			expr = p.finishCall(expr)

		} else if p.match(DOT) {
			name := p.consume(IDENTIFIER, "Expected property name after '.'.")
			expr = &GetExpr{name: name, object: expr}

		} else {
			break
		}
	}
	return expr
}

func (p *Parser) finishCall(callee Expr) Expr {
	args := []Expr{}
	if !p.check(PAREN_RIGHT) {
		for {
			if len(args) > 255 {
				parseErrf("Can't have more than 255 arguments.")
			}
			args = append(args, p.parseExpr())
			if !p.match(COMMA) {
				break
			}
		}
	}
	paren := p.consume(PAREN_RIGHT, "Expected closing ')' after call args.")

	return &Call{callee: callee, paren: paren, args: args}
}

func (p *Parser) parsePrimary() Expr {
	switch {
	case p.match(FALSE):
		return &Literal{val: false}
	case p.match(TRUE):
		return &Literal{val: true}
	case p.match(NIL):
		return &Literal{val: nil}
	case p.match(STRING):
		return &Literal{val: p.previous().Literal}
	case p.match(NUMBER):
		// The book parses floats in the scanner.
		f, _ := strconv.ParseFloat(p.previous().Literal, 64)
		return &Literal{val: f}
	case p.match(PAREN_LEFT):
		expr := p.parseExpr()
		p.consume(PAREN_RIGHT, "Expected closing ')'")
		return &Grouping{group: expr}
	case p.match(IDENTIFIER):
		return &Variable{name: p.previous()}
	case p.match(THIS):
		return &ThisExpr{keyword: p.previous()}
	case p.match(SUPER):
		keyword := p.previous()
		p.consume(DOT, "Expected '.' after super keyword.")
		method := p.consume(IDENTIFIER, "Expected method name for super invocation.")
		return &SuperExpr{keyword: keyword, method: method}
	default:
		at := p.peek()
		p.error(at.Line, "Expected expression")
		p.sync()
		return nil
	}
}

// Check if the next token has type tt.
// Does not move forward.
func (p *Parser) check(tt TokenType) bool {
	return p.peek().Kind == tt
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

func (p *Parser) consume(tt TokenType, msg string) Token {
	at := p.peek()
	if at.Kind != tt {
		p.error(at.Line, msg)
		p.sync()
		return Token{Kind: ILLEGAL, Line: at.Line}
	}

	return p.advance()
}

func (p *Parser) error(line int, msg string) {
	// Emulate exceptions, unwinding the stack.
	parseErrf("error on line %d: %s", line, msg)
}

// FIXME: This will probably invalidate expectations up the stack?
// But it should guarantee to make some progress, else we're stuck.
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
