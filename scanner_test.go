package glox_test

import (
	"testing"

	"github.com/vikblom/glox"
)

func TestScannerKinds(t *testing.T) {
	tests := []struct {
		src  string
		want glox.TokenType
	}{
		// Single char
		{"", glox.EOF},
		{"(", glox.PAREN_LEFT},
		{")", glox.PAREN_RIGHT},
		{"{", glox.BRACE_LEFT},
		{"}", glox.BRACE_RIGHT},
		{",", glox.COMMA},
		{".", glox.DOT},
		{"-", glox.DASH},
		{"+", glox.PLUS},
		{"*", glox.STAR},
		{"/", glox.SLASH},
		{";", glox.SEMICOLON},

		// 1 or 2 chars
		{"/", glox.SLASH},
		{"//", glox.COMMENT},
		{"!", glox.BANG},
		{"!=", glox.BANG_EQUAL},
		{"=", glox.EQUAL},
		{"==", glox.EQUAL_EQUAL},
		{"<", glox.LESS},
		{"<=", glox.LESS_EQUAL},
		{">", glox.GREATER},
		{">=", glox.GREATER_EQUAL},

		{`""`, glox.STRING},
		{`"foo"`, glox.STRING},

		// Multiple chars
		{"1", glox.NUMBER},
		{"1.23", glox.NUMBER},
		{"0.23", glox.NUMBER},

		{"foo", glox.IDENTIFIER},
		{"_foo", glox.IDENTIFIER},

		// Keywords
		{"and", glox.AND},
		{"class", glox.CLASS},
		{"else", glox.ELSE},
		{"false", glox.FALSE},
		{"fun", glox.FUN},
		{"for", glox.FOR},
		{"if", glox.IF},
		{"nil", glox.NIL},
		{"or", glox.OR},
		{"print", glox.PRINT},
		{"return", glox.RETURN},
		{"super", glox.SUPER},
		{"this", glox.THIS},
		{"true", glox.TRUE},
		{"var", glox.VAR},
		{"while", glox.WHILE},
	}

	for _, tt := range tests {
		got := glox.NewScanner([]byte(tt.src)).Scan()
		if got.Kind != tt.want {
			t.Errorf("Scanner(%q).Scan() = %q but want %q)", tt.src, got.Kind, tt.want)
		}
	}
}

func TestScannerSkipsWhitespace(t *testing.T) {
	tests := []struct {
		src  string
		want glox.TokenType
	}{
		{"", glox.EOF},
		{"  ", glox.EOF},
		{" \n\n  ", glox.EOF},
		{"  (", glox.PAREN_LEFT},
		{"  \n(", glox.PAREN_LEFT},
		{"  \n()", glox.PAREN_LEFT},
	}

	for _, tt := range tests {
		got := glox.NewScanner([]byte(tt.src)).Scan()
		if got.Kind != tt.want {
			t.Errorf("Scanner(%q).Scan() = %s but want %s)", tt.src, got.Kind, tt.want)
		}
	}
}

func TestScannerTokens(t *testing.T) {
	tests := []struct {
		src  string
		want glox.Token
	}{
		{
			src: "{",
			want: glox.Token{
				Kind:    glox.BRACE_LEFT,
				Line:    1,
				Literal: "{",
			},
		},
		{
			src: "\n{",
			want: glox.Token{
				Kind:    glox.BRACE_LEFT,
				Line:    2,
				Literal: "{",
			},
		},
		{
			src: "// foo",
			want: glox.Token{
				Kind:    glox.COMMENT,
				Line:    1,
				Literal: "// foo",
			},
		},
		{
			src: "\n    // foo bar\n",
			want: glox.Token{
				Kind:    glox.COMMENT,
				Line:    2,
				Literal: "// foo bar",
			},
		},
		{
			src: `"foo"`,
			want: glox.Token{
				Kind:    glox.STRING,
				Line:    1,
				Literal: `"foo"`,
			},
		},
		{
			// Multiline string.
			src: `"foo
bar"`,
			want: glox.Token{
				Kind: glox.STRING,
				Line: 1,
				Literal: `"foo
bar"`,
			},
		},
		{
			src: "1.23",
			want: glox.Token{
				Kind:    glox.NUMBER,
				Line:    1,
				Literal: "1.23",
			},
		},
		{
			// Lox does not allow trailing period, so
			// this will be two tokens, number & dot.
			src: "123.",
			want: glox.Token{
				Kind:    glox.NUMBER,
				Line:    1,
				Literal: "123",
			},
		},
		{
			// Lox does not allow leading period
			// this will be two tokens, dot and number.
			src: ".123",
			want: glox.Token{
				Kind:    glox.DOT,
				Line:    1,
				Literal: ".",
			},
		},
		{
			src: "foo",
			want: glox.Token{
				Kind:    glox.IDENTIFIER,
				Line:    1,
				Literal: `foo`,
			},
		},
		{
			src: "_foo",
			want: glox.Token{
				Kind:    glox.IDENTIFIER,
				Line:    1,
				Literal: "_foo",
			},
		},
	}

	for _, tt := range tests {
		got := glox.NewScanner([]byte(tt.src)).Scan()
		if got != tt.want {
			t.Errorf("Scanner(%q).Scan()\ngot:  %s\nwant: %s)", tt.src, got.String(), tt.want.String())
		}
	}
}

func TestScanUnclosedString(t *testing.T) {
	sc := glox.NewScanner([]byte(`"foo`))
	got := sc.Scan()

	if got.Kind != glox.ILLEGAL {
		t.Fatalf("parsing unclosed string should be ILLEGAL, but got: %s", got.Kind)
	}
}

func TestScanMany(t *testing.T) {
	src := []byte(`
// this is a comment
(( )){}
!*+-/=<> <= ==
`)
	sc := glox.NewScanner(src)

	toks := []glox.Token{}
	for {
		tok := sc.Scan()
		if tok.Kind == glox.ILLEGAL {
			t.Fatalf("ILLEGAL token encountered: %+v", tok)
		}
		if tok.Kind == glox.EOF {
			break
		}
		toks = append(toks, tok)
	}

	if len(toks) != 17 {
		t.Errorf("expected 17 tokens but got %d in:\n%q", len(toks), src)
	}
}
