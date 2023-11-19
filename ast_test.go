package glox_test

import (
	"testing"

	"github.com/vikblom/glox"
)

func TestPrinter(t *testing.T) {
	tests := []struct {
		src, want string
	}{
		{src: "1 + 1;", want: "(expr (+ 1 1))"},
		{src: "1 + (2 - 3);", want: "(expr (+ 1 (group (- 2 3))))"},
		{src: "-1 * -1;", want: "(expr (* (- 1) (- 1)))"},
		{src: "var a;", want: "(var a <nil>)"},
		{src: "var a = 1;", want: "(var a 1)"},
		{src: `var a = "foo";`, want: `(var a "foo")`},
		{src: `a = "foo";`, want: `(expr (assign a "foo"))`},
		{src: `{ a = 1; b = 2; }`, want: `(block (expr (assign a 1)) (expr (assign b 2)))`},

		{src: `if (1 == 2) print 1;`, want: `(if (== 1 2) then (print 1))`},
		{src: `if (1 == 2) print 1; else print 2;`, want: `(if (== 1 2) then (print 1) else (print 2))`},

		// or binds lower than and.
		{src: `if (a and b or c) 1;`, want: `(if (or (and a b) c) then (expr 1))`},
		{src: `if (a or b and c) 1;`, want: `(if (or a (and b c)) then (expr 1))`},
	}

	for _, tt := range tests {
		toks, err := glox.ScanString(tt.src)
		if err != nil {
			t.Fatalf("scan string: %s", err)
		}

		parser := glox.NewParser(toks)
		stmts, err := parser.Parse()
		if err != nil {
			t.Fatalf("parse: %s", err)
		}

		got := glox.PrintAST(stmts[0]) // FIXME
		if tt.want != got {
			t.Errorf(`PrintAst("%s") = %q but want %q`, tt.src, got, tt.want)
		}
	}
}
