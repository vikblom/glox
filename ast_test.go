package glox_test

import (
	"testing"

	"github.com/vikblom/glox"
)

func TestPrinter(t *testing.T) {

	tests := []struct {
		src, want string
	}{
		{src: "1 + 1;", want: "(+ 1 1)"},
		{src: "1 + (2 - 3);", want: "(+ 1 (group (- 2 3)))"},
		{src: "-1 * -1;", want: "(* (- 1) (- 1))"},
	}

	for _, tt := range tests {
		toks, err := glox.ScanString(tt.src)
		if err != nil {
			t.Fatalf("scan string: %s", err)
		}

		parser := glox.NewParser(toks)
		expr, err := parser.Parse()
		if err != nil {
			t.Fatalf("parse: %s", err)
		}

		got := glox.PrintAST(expr)
		if tt.want != got {
			t.Fatalf(`PrintAst("%s") = %q but want %q`, tt.src, got, tt.want)
		}
	}
}
