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
			Kind: glox.EOF,
		},
	}

	p := glox.NewParser(toks)
	exp, err := p.Parse()
	if err != nil {
		t.Fatalf("parsing failed: %s", err)
	}

	t.Logf("%#v", exp)
}
