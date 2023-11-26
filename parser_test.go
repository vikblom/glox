package glox_test

import (
	"testing"

	"github.com/vikblom/glox"
)

func TestParser(t *testing.T) {
	toks := []glox.Token{
		{
			Kind:    glox.NUMBER,
			Literal: "1",
		},
		{
			Kind:    glox.EQUAL_EQUAL,
			Literal: "==",
		},
		{
			Kind:    glox.NUMBER,
			Literal: "1",
		},
		{
			Kind:    glox.SEMICOLON,
			Literal: ";",
		},
		{
			Kind: glox.EOF,
		},
	}

	p := glox.NewParser(toks)
	exp, err := p.Parse()
	if err != nil {
		t.Fatalf("parsing failed: %s", err)
	}
	_ = exp
}

func TestParseSyntaxError(t *testing.T) {
	src := "1 + ;"
	toks, err := glox.ScanString(src)
	if err != nil {
		t.Fatalf("scan string %q: %s", src, err)
	}

	p := glox.NewParser(toks)
	_, err = p.Parse()
	if err == nil {
		t.Fatalf("expected parse to fail but got: %s", err)
	}

}
