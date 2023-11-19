package glox_test

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/vikblom/glox"
	"golang.org/x/tools/txtar"
)

var updateGolden = flag.Bool("golden", false, "Update golden files")

func TestTestdata(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.txt")
	if len(files) == 0 {
		t.Fatalf("no testdata")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			a, err := txtar.ParseFile(file)
			if err != nil {
				t.Fatalf("txtar parse: %s", err)
			}
			if len(a.Files) != 2 || (a.Files[0].Name != "src.lox") || (a.Files[1].Name != "stdout") {
				t.Fatalf("%s: want two files named \"src.lox\" & \"stdout\"", file)
			}

			src := a.Files[0].Data
			toks, err := glox.ScanBytes(src)
			if err != nil {
				t.Fatalf("scan string: %s", err)
			}

			parser := glox.NewParser(toks)
			stmts, err := parser.Parse()
			if err != nil {
				t.Fatalf("parse: %s", err)
			}

			buf := bytes.NewBuffer(nil)
			i := glox.NewInterpreter(buf)
			err = i.Interpret(stmts)
			if err != nil {
				t.Fatalf("interpret: %s", err)
			}
			got := buf.String()

			if *updateGolden {
				a.Files[1].Data = buf.Bytes()
				bs := txtar.Format(a)
				os.WriteFile(file, bs, 0644)
				return
			}

			want := string(a.Files[1].Data)
			if d := cmp.Diff(want, got); d != "" {
				t.Fatalf("interpreted stdout diff (-want, +got):\n%s", d)
			}
		})
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

		i := glox.NewInterpreter(io.Discard)
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

		i := glox.NewInterpreter(io.Discard)
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

func TestEvalPrints(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{src: "var a; print a;", want: "<nil>\n"},
		{src: "var a = 1; print a;", want: "1\n"},
		{src: `var hello = 1; print "hello";`, want: "\"hello\"\n"},
		{src: `var a = 1; var b = 2; print a + b;`, want: "3\n"},
		{src: `var a = 1; a = 2; print a;`, want: "2\n"},
		// Assignment is an expression.
		{src: `var a = 1; print a = 123;`, want: "123\n"},
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

		buf := bytes.NewBuffer(nil)
		i := glox.NewInterpreter(buf)
		err = i.Interpret(stmts)
		if err != nil {
			t.Fatalf("interpret: %s", err)
		}

		got := buf.String()
		if tt.want != got {
			t.Fatalf(`Interpret("%s") = %q but want %q`, tt.src, got, tt.want)
		}
	}
}
