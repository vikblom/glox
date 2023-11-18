package glox

import (
	"fmt"
	"strings"
)

// Visitor can visit any Expr.
//
// The book makes it more explicit what needs to be handled.
// But that seems tedious.
//
//	type ExprVisitor interface {
//		VisitLiteralExpr(Literal) any
//		VisitUnaryExpr(UnaryExpr) any
//		VisitBinaryExpr(BinaryExpr) any
//	}
type Visitor func(Expr) any

type Expr interface {
	// Visitable.
	Accept(Visitor) any
}

type (
	BinaryExpr struct {
		op          Token
		left, right Expr
	}

	UnaryExpr struct {
		op    Token
		right Expr
	}

	Literal struct {
		val any
	}

	Grouping struct {
		group Expr
	}
)

func (e *BinaryExpr) Accept(v Visitor) any { return v(e) }
func (e *UnaryExpr) Accept(v Visitor) any  { return v(e) }
func (e *Literal) Accept(v Visitor) any    { return v(e) }
func (e *Grouping) Accept(v Visitor) any   { return v(e) }

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

// EvalAST rooted at expr.
// There are 4 types used for values: any, string, float64 & bool.
func EvalAST(expr Expr) (v any, err error) {
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(runtimeError); ok {
				err = re.error
			} else {
				panic(r)
			}
		}
	}()

	v = expr.Accept(evalVisitor)

	return
}

func evalVisitor(e Expr) any {
	switch v := e.(type) {
	case *Grouping:
		return evalVisitor(v.group)

	case *BinaryExpr:
		l := evalVisitor(v.left)
		r := evalVisitor(v.right)
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
			r := evalVisitor(v.right)
			mustBeNumbers(v.op, r)
			if f, ok := r.(float64); ok {
				return -f
			}
		case BANG:
			vv := evalVisitor(v.right)
			return !isTruthy(vv)
		}
		runtimeErrf("impossible unary")

	case *Literal:
		return v.val

	default:
		panic(fmt.Sprintf("unknown as node: %T :: %#v", e, e))
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

// PrintAST representation of Expr node.
func PrintAST(expr Expr) string {
	return expr.Accept(printVisitor).(string)
}

func printVisitor(e Expr) any {
	switch v := e.(type) {
	case *BinaryExpr:
		l := printVisitor(v.left)
		r := printVisitor(v.right)
		return parenthesize(v.op.Literal, l, r)
	case *UnaryExpr:
		r := printVisitor(v.right)
		return parenthesize(v.op.Literal, r)
	case *Literal:
		return v.val // TODO: Parenthesis?
	case *Grouping:
		g := printVisitor(v.group)
		return parenthesize("group", g)
	default:
		panic(fmt.Sprintf("unknown as node: %T :: %#v", e, e))
	}
}

func parenthesize(vs ...any) string {
	if len(vs) == 0 {
		return "()"
	}
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "(")

	fmt.Fprintf(&sb, "%s", vs[0])
	for _, v := range vs[1:] {
		fmt.Fprintf(&sb, " %s", v)
	}
	fmt.Fprintf(&sb, ")")

	return sb.String()
}