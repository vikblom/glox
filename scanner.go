package glox

import (
	"fmt"
)

type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	PAREN_LEFT
	PAREN_RIGHT
	BRACE_LEFT
	BRACE_RIGHT
	COMMA
	DOT
	DASH
	PLUS
	SEMICOLON
	SLASH
	STAR

	BANG
	BANG_EQUAL
	EQUAL
	EQUAL_EQUAL
	GREATER
	GREATER_EQUAL
	LESS
	LESS_EQUAL

	IDENTIFIER
	STRING
	NUMBER

	AND
	CLASS
	ELSE
	FALSE
	FUN
	FOR
	IF
	NIL
	OR
	PRINT
	RETURN
	SUPER
	THIS
	TRUE
	VAR
	WHILE
)

var tokenTypes = map[TokenType]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	COMMENT: "COMMENT",

	PAREN_LEFT:  "(",
	PAREN_RIGHT: ")",
	BRACE_LEFT:  "{",
	BRACE_RIGHT: "}",
	COMMA:       ",",
	DOT:         ".",
	DASH:        "-",
	PLUS:        "+",
	SEMICOLON:   ";",
	SLASH:       "/",
	STAR:        "*",

	BANG:          "!",
	BANG_EQUAL:    "!=",
	EQUAL:         "=",
	EQUAL_EQUAL:   "==",
	GREATER:       ">",
	GREATER_EQUAL: ">=",
	LESS:          "<",
	LESS_EQUAL:    "<=",

	IDENTIFIER: "IDENT",
	STRING:     "STRING",
	NUMBER:     "NUMBER",

	AND:    "",
	CLASS:  "",
	ELSE:   "",
	FALSE:  "",
	FUN:    "",
	FOR:    "",
	IF:     "",
	NIL:    "",
	OR:     "",
	PRINT:  "",
	RETURN: "",
	SUPER:  "",
	THIS:   "",
	TRUE:   "",
	VAR:    "",
	WHILE:  "",
}

var keywords = map[string]TokenType{
	"and":    AND,
	"class":  CLASS,
	"else":   ELSE,
	"false":  FALSE,
	"fun":    FUN,
	"for":    FOR,
	"if":     IF,
	"nil":    NIL,
	"or":     OR,
	"print":  PRINT,
	"return": RETURN,
	"super":  SUPER,
	"this":   THIS,
	"true":   TRUE,
	"var":    VAR,
	"while":  WHILE,
}

func (t TokenType) String() string {
	return tokenTypes[t]
}

type Token struct {
	Kind TokenType
	// Lox smuggles values for strings and numbers in the lexeme field.
	// Go just passses the Literal value.
	Literal string

	Line int
}

func (t *Token) String() string {
	return fmt.Sprintf("[%d] %s: %q", t.Line, t.Kind, t.Literal)
}

func isDigit(b byte) bool        { return '0' <= b && b <= '9' }
func isAlpha(b byte) bool        { return 'a' <= b && b <= 'z' || 'A' <= b && b < 'Z' || b == '_' }
func isAlphaNumeric(b byte) bool { return isDigit(b) || isAlpha(b) }

// Scanner inspired by Crafting Interpreters and Go.
type Scanner struct {
	// Can only be scanned once, which might be a problem?
	// A []byte could be nicer.
	src []byte

	// Scanner state.
	// at the next byte to read.
	at int
	// line of at, starting from 1.
	line int
}

func NewScanner(src []byte) *Scanner {
	return &Scanner{src: src, line: 1}
}

func (s *Scanner) advance() byte {
	b := s.src[s.at]
	if b == '\n' {
		s.line += 1
	}
	s.at += 1
	return b
}

func (s *Scanner) peek() byte {
	if s.finished() {
		return byte(0)
	}
	return s.src[s.at]
}

func (s *Scanner) peekpeek() byte {
	if s.at+1 >= len(s.src) {
		return byte(0)
	}
	return s.src[s.at+1]
}

func (s *Scanner) skip() {
	if s.finished() {
		return
	}
	if s.src[s.at] == '\n' {
		s.line += 1
	}
	s.at += 1
}

// consume b if it is the next byte to scan.
func (s *Scanner) consume(b byte) bool {
	if s.peek() != b {
		return false
	}
	s.skip()
	return true
}

// finished if there are no more bytes.
func (s *Scanner) finished() bool {
	return s.at >= len(s.src)
}

func (s *Scanner) Scan() Token {
	// Skip whitespace so s is at some non-whitespace byte.
whitespace:
	for {
		switch s.peek() {
		case ' ', '\n', '\r', '\t':
			s.skip()
		default:
			break whitespace
		}
	}
	if s.finished() {
		return Token{Kind: EOF, Line: s.line}
	}

	start := s.at
	line := s.line

	var kind TokenType
	b := s.advance()
	switch {
	case isDigit(b):
		kind = NUMBER
		for isDigit(s.peek()) {
			s.skip()
		}
		if s.peek() == '.' && isDigit(s.peekpeek()) {
			s.skip() // eat the .
			for isDigit(s.peek()) {
				s.skip()
			}
		}

	case isAlpha(b):
		kind = IDENTIFIER
		for isAlphaNumeric(s.peek()) {
			s.skip()
		}

		if k, ok := keywords[string(s.src[start:s.at])]; ok {
			kind = k
		}

	default:
		switch b {
		case '(':
			kind = PAREN_LEFT
		case ')':
			kind = PAREN_RIGHT
		case '{':
			kind = BRACE_LEFT
		case '}':
			kind = BRACE_RIGHT
		case ',':
			kind = COMMA
		case '.':
			kind = DOT
		case '-':
			kind = DASH
		case '+':
			kind = PLUS
		case ';':
			kind = SEMICOLON
		case '*':
			kind = STAR

		case '!':
			if s.consume('=') {
				kind = BANG_EQUAL
			} else {
				kind = BANG
			}
		case '=':
			if s.consume('=') {
				kind = EQUAL_EQUAL
			} else {
				kind = EQUAL
			}
		case '<':
			if s.consume('=') {
				kind = LESS_EQUAL
			} else {
				kind = LESS
			}
		case '>':
			if s.consume('=') {
				kind = GREATER_EQUAL
			} else {
				kind = GREATER
			}

		case '/':
			if s.consume('/') {
				kind = COMMENT
				for s.peek() != '\n' && !s.finished() {
					s.skip()
				}
			} else {
				kind = SLASH
			}

		case '"':
			kind = STRING
			for s.peek() != '"' && !s.finished() {
				s.skip()
			}
			if s.finished() {
				// TODO: Error handling.
				// TODO: Indicate upstream that we want more data?
				// Would be nice when running as an interpreter.
				// Mutliline-strings, functions etc. Needs to work both here and in the parser.
				kind = ILLEGAL
			} else {
				s.skip() // closing "
			}

		default:
			// Unexpected character.
			kind = ILLEGAL
		}
	}

	// TODO: Test Line and Literal.
	return Token{Kind: kind, Line: line, Literal: string(s.src[start:s.at])}
}

func ScanString(s string) ([]Token, error) {
	sc := NewScanner([]byte(s))

	toks := []Token{}
	for {
		tok := sc.Scan()
		if tok.Kind == ILLEGAL {
			return nil, fmt.Errorf("ILLEGAL token encountered: %+v", tok)
		}
		if tok.Kind == EOF {
			break
		}
		toks = append(toks, tok)
	}
	return toks, nil
}
