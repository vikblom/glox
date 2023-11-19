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
		stmts, err := parser.Parse()
		if err != nil {
			t.Fatalf("parse: %s", err)
		}

		got := glox.PrintAST(stmts[0].Stmt()) // FIXME
		if tt.want != got {
			t.Errorf(`PrintAst("%s") = %q but want %q`, tt.src, got, tt.want)
		}
	}
}

func TestEvalArithmetic(t *testing.T) {
	tests := []struct {
		src  string
		want float64
	}{
		{src: "1;", want: 1.0},
		{src: "1 + 1;", want: 2.0},
		{src: "1 + (2 - 3);", want: 0.0},
		{src: "-1 * -1;", want: 1.0},
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

		i := glox.NewInterpreter()
		v, err := i.EvalAST(stmts[0].Stmt()) // FIXME
		if err != nil {
			t.Fatalf("eval ast: %s", err)
		}
		got, ok := v.(float64)
		if !ok {
			t.Fatalf("expected float64 but got %T :: %v", v, v)
		}
		if tt.want != got {
			t.Fatalf(`PrintAst("%s") = %f but want %f`, tt.src, got, tt.want)
		}
	}
}

func TestEvalComparison(t *testing.T) {

	tests := []struct {
		src  string
		want bool
	}{
		{src: "1 < 2;", want: true},
		{src: "(1 + 1) <= (1 * 2);", want: true},
		{src: "1 + (2 - 3) == 0;", want: true},
		{src: "true == true;", want: true},
		{src: "true != true;", want: false},

		{src: "1 >= 3;", want: false},
		{src: "!false;", want: true},
		{src: "!(false) == true;", want: true},
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

		i := glox.NewInterpreter()
		v, err := i.EvalAST(stmts[0].Stmt()) // FIXME
		if err != nil {
			t.Fatalf("eval ast: %s", err)
		}
		got, ok := v.(bool)
		if !ok {
			t.Fatalf("expected bool but got %T :: %v", v, v)
		}
		if tt.want != got {
			t.Fatalf(`PrintAst("%s") = %v but want %v`, tt.src, got, tt.want)
		}
	}
}
