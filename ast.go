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

	FuncStmt struct {
		name   Token
		params []Token
		// Does this need to be a slice?
		body []Stmt
	}

	VarStmt struct {
		name Token
		init Expr
	}

	BlockStmt struct {
		statements []Stmt
	}

	IfStmt struct {
		cond                   Expr
		thenBranch, elseBranch Stmt
	}

	WhileStmt struct {
		cond Expr
		body Stmt
	}

	ReturnStmt struct {
		keyword Token
		value   Expr
	}

	ClassStmt struct {
		name    Token
		methods []Stmt
	}
)

func (s *PrintStmt) Accept(v Visitor) any  { return v(s) }
func (s *ExprStmt) Accept(v Visitor) any   { return v(s) }
func (s *FuncStmt) Accept(v Visitor) any   { return v(s) }
func (s *VarStmt) Accept(v Visitor) any    { return v(s) }
func (s *BlockStmt) Accept(v Visitor) any  { return v(s) }
func (s *IfStmt) Accept(v Visitor) any     { return v(s) }
func (s *WhileStmt) Accept(v Visitor) any  { return v(s) }
func (s *ReturnStmt) Accept(v Visitor) any { return v(s) }
func (s *ClassStmt) Accept(v Visitor) any  { return v(s) }

func (s *PrintStmt) Stmt() Expr  { return s.expr }
func (s *ExprStmt) Stmt() Expr   { return s.expr }
func (s *FuncStmt) Stmt() Expr   { return nil }
func (s *VarStmt) Stmt() Expr    { return s.init }
func (s *BlockStmt) Stmt() Expr  { return nil }
func (s *IfStmt) Stmt() Expr     { return nil }
func (s *WhileStmt) Stmt() Expr  { return nil }
func (s *ReturnStmt) Stmt() Expr { return nil }
func (s *ClassStmt) Stmt() Expr  { return nil }

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

	LogicalExpr struct {
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

	Call struct {
		callee Expr
		paren  Token
		args   []Expr
	}

	GetExpr struct {
		object Expr
		name   Token
	}

	SetExpr struct {
		// object.name = value
		object Expr
		name   Token
		value  Expr
	}

	ThisExpr struct {
		keyword Token
	}
)

func (e *BinaryExpr) Accept(v Visitor) any  { return v(e) }
func (e *LogicalExpr) Accept(v Visitor) any { return v(e) }
func (e *UnaryExpr) Accept(v Visitor) any   { return v(e) }
func (e *Literal) Accept(v Visitor) any     { return v(e) }
func (e *Grouping) Accept(v Visitor) any    { return v(e) }
func (e *Variable) Accept(v Visitor) any    { return v(e) }
func (e *Assign) Accept(v Visitor) any      { return v(e) }
func (e *Call) Accept(v Visitor) any        { return v(e) }
func (e *GetExpr) Accept(v Visitor) any     { return v(e) }
func (e *SetExpr) Accept(v Visitor) any     { return v(e) }
func (e *ThisExpr) Accept(v Visitor) any    { return v(e) }

func (e *BinaryExpr) expr()  {}
func (e *LogicalExpr) expr() {}
func (e *UnaryExpr) expr()   {}
func (e *Literal) expr()     {}
func (e *Grouping) expr()    {}
func (e *Variable) expr()    {}
func (e *Assign) expr()      {}
func (e *Call) expr()        {}
func (e *GetExpr) expr()     {}
func (e *SetExpr) expr()     {}
func (e *ThisExpr) expr()    {}

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
	case *LogicalExpr:
		l := printVisitor(v.left)
		r := printVisitor(v.right)
		return parenthesize(v.op.Literal, l, r)
	case *UnaryExpr:
		r := printVisitor(v.right)
		return parenthesize(v.op.Literal, r)
	case *Literal:
		return fmt.Sprintf("%v", v.val) // TODO: Parenthesis?
	case *Variable:
		return v.name.Literal
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
	case *IfStmt:
		vs := []any{"if", printVisitor(v.cond)}
		vs = append(vs, "then", printVisitor(v.thenBranch))
		if v.elseBranch != nil {
			vs = append(vs, "else", printVisitor(v.elseBranch))
		}
		return parenthesize(vs...)
	case *Call:
		vs := []any{"call", printVisitor(v.callee)}
		for _, s := range v.args {
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
