package glox

import "fmt"

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
}

func NewEnv() *Env {
	return &Env{vars: map[string]any{}}
}

func (e *Env) define(name string, val any) {
	e.vars[name] = val
}

type Interpreter struct {
	env *Env
}

func NewInterpreter() *Interpreter {
	return &Interpreter{
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
	v = node.Accept(i.evalVisitor)
	return
}

func (i *Interpreter) evalVisitor(node Node) any {
	switch v := node.(type) {
	case *Grouping:
		return i.evalVisitor(v.group)

	case *BinaryExpr:
		l := i.evalVisitor(v.left)
		r := i.evalVisitor(v.right)
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

	case *UnaryExpr:
		switch v.op.Kind {
		case DASH:
			r := i.evalVisitor(v.right)
			mustBeNumbers(v.op, r)
			if f, ok := r.(float64); ok {
				return -f
			}
		case BANG:
			vv := i.evalVisitor(v.right)
			return !isTruthy(vv)
		}
		runtimeErrf("impossible unary")

	case *Literal:
		return v.val

	case *PrintStmt:
		val := i.evalVisitor(v.expr)
		fmt.Printf("%v\n", val)
		return nil

	case *ExprStmt:
		_ = i.evalVisitor(v.expr)
		return nil

	case *VarStmt:
		var val any
		if v.init != nil {
			val = i.evalVisitor(v.init)
		}
		i.env.define(v.name.Literal, val)
		return nil

	default:
		panic(fmt.Sprintf("unknown as node: %T :: %#v", node, node))
	}

	panic("unreachable")
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
