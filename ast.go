package glox

import (
	"fmt"
	"strings"
)

// Visitor can visit any Expr or Stmt.
//
// The book makes it more explicit what needs to be handled.
// But that seems tedious.
//
//	type ExprVisitor interface {
//		VisitLiteralExpr(Literal) any
//		VisitUnaryExpr(UnaryExpr) any
//		VisitBinaryExpr(BinaryExpr) any
//	}
type Visitor func(Node) any

// Node in the AST which is visitable.
type Node interface {
	Accept(Visitor) any
}

type Stmt interface {
	Node
	// TOOD: Is this needed?
	Stmt() Expr
}

type (
	PrintStmt struct {
		expr Expr
	}

	ExprStmt struct {
		expr Expr
	}

	VarStmt struct {
		name Token
		init Expr
	}

	BlockStmt struct {
		statements []Stmt
	}
)

func (s *PrintStmt) Accept(v Visitor) any { return v(s) }
func (s *ExprStmt) Accept(v Visitor) any  { return v(s) }
func (s *VarStmt) Accept(v Visitor) any   { return v(s) }
func (s *BlockStmt) Accept(v Visitor) any { return v(s) }

func (s *PrintStmt) Stmt() Expr { return s.expr }
func (s *ExprStmt) Stmt() Expr  { return s.expr }
func (s *VarStmt) Stmt() Expr   { return s.init }
func (s *BlockStmt) Stmt() Expr { return nil }

type Expr interface {
	Node
	// TODO: Is this needed?
	expr()
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

	Variable struct {
		name Token
	}

	Assign struct {
		name Token
		val  Expr
	}
)

func (e *BinaryExpr) Accept(v Visitor) any { return v(e) }
func (e *UnaryExpr) Accept(v Visitor) any  { return v(e) }
func (e *Literal) Accept(v Visitor) any    { return v(e) }
func (e *Grouping) Accept(v Visitor) any   { return v(e) }
func (e *Variable) Accept(v Visitor) any   { return v(e) }
func (e *Assign) Accept(v Visitor) any     { return v(e) }

func (e *BinaryExpr) expr() {}
func (e *UnaryExpr) expr()  {}
func (e *Literal) expr()    {}
func (e *Grouping) expr()   {}
func (e *Variable) expr()   {}
func (e *Assign) expr()     {}

// PrintAST representation of Expr node.
func PrintAST(nodes ...Node) string {
	sb := strings.Builder{}
	for _, n := range nodes {
		v := n.Accept(printVisitor)
		s, ok := v.(string)
		if !ok {
			panic(fmt.Sprintf("expected string but got %T :: %v", v, v))
		}
		fmt.Fprintf(&sb, "%s\n", s)
	}
	return strings.TrimSpace(sb.String())
}

func printVisitor(node Node) any {
	switch v := node.(type) {
	case *BinaryExpr:
		l := printVisitor(v.left)
		r := printVisitor(v.right)
		return parenthesize(v.op.Literal, l, r)
	case *UnaryExpr:
		r := printVisitor(v.right)
		return parenthesize(v.op.Literal, r)
	case *Literal:
		return fmt.Sprintf("%v", v.val) // TODO: Parenthesis?
	case *Grouping:
		g := printVisitor(v.group)
		return parenthesize("group", g)
	case *Assign:
		g := printVisitor(v.val)
		return parenthesize("assign", v.name.Literal, g)

	case *PrintStmt:
		r := printVisitor(v.expr)
		return parenthesize("print", r)
	case *ExprStmt:
		e := printVisitor(v.expr)
		return parenthesize("expr", e)
	case *VarStmt:
		vv := "<nil>"
		if v.init != nil {
			vv = printVisitor(v.init).(string)
		}
		return parenthesize("var", v.name.Literal, vv)
	case *BlockStmt:
		vs := []any{"block"}
		for _, s := range v.statements {
			vs = append(vs, printVisitor(s))
		}
		return parenthesize(vs...)
	default:
		panic(fmt.Sprintf("unknown as node: %T :: %#v", node, node))
	}
}

func parenthesize(vs ...any) string {
	if len(vs) == 0 {
		return "()"
	}
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "(")

	fmt.Fprintf(&sb, "%v", vs[0])
	for _, v := range vs[1:] {
		fmt.Fprintf(&sb, " %v", v)
	}
	fmt.Fprintf(&sb, ")")

	return sb.String()
}
