package glox

import (
	"fmt"
	"io"
)

type runtimeError struct{ error }

func runtimeErrf(format string, args ...any) {
	panic(runtimeError{error: fmt.Errorf(format, args...)})
}

func mustBeNumbers(tok Token, args ...any) {
	for _, o := range args {
		if _, ok := o.(float64); !ok {
			runtimeErrf("%q requires number arguments: %T", tok.Literal, o)
		}
	}
}

type Env struct {
	vars map[string]any
	// Parent environment.
	enclosing *Env
}

func NewEnv() *Env {
	return &Env{
		vars:      map[string]any{},
		enclosing: nil,
	}
}

// Fork e into a child Env.
func (e *Env) Fork() *Env {
	return &Env{
		vars:      map[string]any{},
		enclosing: e,
	}
}

func (e *Env) define(name string, val any) {
	e.vars[name] = val
}

func (e *Env) assign(name string, val any) {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = val
		return
	}
	if e.enclosing != nil {
		e.enclosing.assign(name, val)
		return
	}
	runtimeErrf("undefined %q", name)
}

func (e *Env) retrieve(name string) any {
	v, ok := e.vars[name]
	if ok {
		return v
	}

	if e.enclosing != nil {
		return e.enclosing.retrieve(name)
	}

	runtimeErrf("undefined %q", name)
	return nil
}

type Interpreter struct {
	out io.Writer
	env *Env
}

func NewInterpreter(out io.Writer) *Interpreter {
	return &Interpreter{
		out: out,
		env: NewEnv(),
	}
}

func (i *Interpreter) Interpret(stmts []Stmt) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(runtimeError); ok {
				err = re.error
			} else {
				panic(r)
			}
		}
	}()

	for _, s := range stmts {
		_, err := i.EvalAST(s)
		if err != nil {
			return fmt.Errorf("runtime error: %w", err)
		}
	}
	return nil
}

// EvalAST rooted at node.
// There are 4 types used for values: any, string, float64 & bool.
func (i *Interpreter) EvalAST(node Node) (v any, err error) {
	v = node.Accept(i.execute)
	return
}

// execute node using this AST visitor function.
func (i *Interpreter) execute(node Node) any {
	switch v := node.(type) {
	case *Grouping:
		return i.execute(v.group)

	case *BinaryExpr:
		l := i.execute(v.left)
		r := i.execute(v.right)
		switch v.op.Kind {
		case EQUAL_EQUAL:
			return isEqual(l, r)
		case BANG_EQUAL:
			return !isEqual(l, r)
		}

		switch v.op.Kind {
		case PLUS:
			mustBeNumbers(v.op, l, r)
			return l.(float64) + r.(float64)
		case DASH:
			mustBeNumbers(v.op, l, r)
			return l.(float64) - r.(float64)
		case STAR:
			mustBeNumbers(v.op, l, r)
			return l.(float64) * r.(float64)
		case SLASH:
			mustBeNumbers(v.op, l, r)
			return l.(float64) / r.(float64)

		case GREATER:
			mustBeNumbers(v.op, l, r)
			return l.(float64) > r.(float64)
		case GREATER_EQUAL:
			mustBeNumbers(v.op, l, r)
			return l.(float64) >= r.(float64)
		case LESS:
			mustBeNumbers(v.op, l, r)
			return l.(float64) < r.(float64)
		case LESS_EQUAL:
			mustBeNumbers(v.op, l, r)
			return l.(float64) <= r.(float64)
		}
		runtimeErrf("impossible binary")

	case *LogicalExpr:
		left := i.execute(v.left)

		// The value of left can short circuit the expression.
		switch v.op.Kind {
		case OR:
			if isTruthy(left) {
				return left
			}
		case AND:
			if !isTruthy(left) {
				return left
			}
		default:
			panic("impossible logical")
		}

		return i.execute(v.right)

	case *UnaryExpr:
		switch v.op.Kind {
		case DASH:
			r := i.execute(v.right)
			mustBeNumbers(v.op, r)
			if f, ok := r.(float64); ok {
				return -f
			}
		case BANG:
			vv := i.execute(v.right)
			return !isTruthy(vv)
		}
		runtimeErrf("impossible unary")

	case *Literal:
		return v.val

	case *Variable:
		return i.env.retrieve(v.name.Literal)

	case *Assign:
		val := i.execute(v.val)
		i.env.assign(v.name.Literal, val)
		return val

	case *PrintStmt:
		val := i.execute(v.expr)
		fmt.Fprintf(i.out, "%v\n", val)
		return nil

	case *ExprStmt:
		_ = i.execute(v.expr)
		return nil

	case *VarStmt:
		var val any
		if v.init != nil {
			val = i.execute(v.init)
		}
		i.env.define(v.name.Literal, val)
		return nil

	case *BlockStmt:
		i.executeBlock(v)
		return nil

	case *IfStmt:
		if isTruthy(i.execute(v.cond)) {
			i.execute(v.thenBranch)
		} else if v.elseBranch != nil {
			i.execute(v.elseBranch)
		}
		return nil

	default:
		panic(fmt.Sprintf("unknown node: %T :: %#v", node, node))
	}

	panic("unreachable")
}

func (i *Interpreter) executeBlock(block *BlockStmt) {
	prev := i.env
	i.env = i.env.Fork()
	defer func() { i.env = prev }()

	for _, s := range block.statements {
		i.execute(s)
	}
}

func isEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		return false
	}
	return a == b // Does this work on interfaces?
}

// Literal false and nil are falsy.
func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}
