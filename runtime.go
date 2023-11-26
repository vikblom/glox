package glox

import (
	"fmt"
	"io"
)

type callable interface {
	call(i *Interpreter, args []any) any
	arity() int
}

type function struct {
	decl    *FuncStmt
	closure *Env
}

func (f *function) arity() int {
	return len(f.decl.params)
}

func (f *function) call(i *Interpreter, args []any) (ret any) {
	// Using panics to unwind the stack on return...
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(returnValue); ok {
				ret = re.any
			} else {
				panic(r)
			}
		}
	}()
	// Each function captures the environment where it was _declared_.
	// Closing over variables there.
	env := f.closure.Fork()
	for i, param := range f.decl.params {
		env.define(param.Literal, args[i])
	}

	i.executeBlock(f.decl.body, env)
	return nil
}

func (f *function) String() string {
	return fmt.Sprintf("<fn %s>", f.decl.name.Literal)
}

type runtimeError struct{ error }

func runtimeErrf(format string, args ...any) {
	panic(runtimeError{error: fmt.Errorf("RUNTIME ERROR: "+format, args...)})
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

func (i *Interpreter) lookupVariable(name Token, expr Expr) any {
	distance, ok := i.locals[expr]
	if !ok {
		return i.global.get(name.Literal)
	}
	return i.scope.up(distance).get(name.Literal)
}

func (e *Env) get(name string) any {
	v, ok := e.vars[name]
	if !ok {
		runtimeErrf("undefined %q", name)
		return nil
	}
	return v
}

func (e *Env) up(distance int) *Env {
	env := e
	for i := 0; i < distance; i++ {
		env = env.enclosing
	}
	return env
}

// returnValue by panic...
type returnValue struct{ any }

type Interpreter struct {
	out    io.Writer
	global *Env
	scope  *Env

	// Static analysis.
	locals map[Expr]int
}

func NewInterpreter(out io.Writer) *Interpreter {
	g := NewEnv()
	g.define("clock", &builtinClock{})

	return &Interpreter{
		out: out,
		// Fixed ref to top level scope.
		global: g,
		// Current scope, will change as we execute.
		scope: g,

		locals: map[Expr]int{},
	}
}

func (i *Interpreter) resolve(expr Expr, depth int) {
	i.locals[expr] = depth
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

	// Statically analyze variable decl/define.
	// TODO: Move somewhere outside?
	r := NewResolver(i)
	for _, s := range stmts {
		r.resolve(s)
	}

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
		return i.lookupVariable(v.name, v)

	case *Assign:
		val := i.execute(v.val)

		dist, ok := i.locals[v] // FIXME: Must this be the Expr?
		if !ok {
			i.global.assign(v.name.Literal, val)
		} else {
			i.scope.up(dist).assign(v.name.Literal, val)
		}

		return val

	case *Call:
		callee := i.execute(v.callee)

		args := []any{}
		for _, a := range v.args {
			args = append(args, i.execute(a))
		}

		callable, ok := callee.(callable)
		if !ok {
			runtimeErrf("Not callable %T", callee)
			return nil
		}
		if callable.arity() != len(args) {
			runtimeErrf("Expected %d arguments but got %d", callable.arity(), len(args))
			return nil
		}
		return callable.call(i, args)

	case *PrintStmt:
		val := i.execute(v.expr)
		fmt.Fprintf(i.out, "%v\n", val)
		return nil

	case *ExprStmt:
		_ = i.execute(v.expr)
		return nil

	case *FuncStmt:
		fn := &function{
			decl:    v,
			closure: i.scope,
		}
		i.scope.define(v.name.Literal, fn)
		return nil

	case *VarStmt:
		var val any
		if v.init != nil {
			val = i.execute(v.init)
		}
		i.scope.define(v.name.Literal, val)
		return nil

	case *BlockStmt:
		i.executeBlock(v.statements, i.scope.Fork())
		return nil

	case *IfStmt:
		if isTruthy(i.execute(v.cond)) {
			i.execute(v.thenBranch)
		} else if v.elseBranch != nil {
			i.execute(v.elseBranch)
		}
		return nil

	case *WhileStmt:
		for isTruthy(i.execute(v.cond)) {
			i.execute(v.body)
		}
		return nil

	case *ReturnStmt:
		var value any
		if v.value != nil {
			value = i.execute(v.value)
		}
		panic(returnValue{value})

	default:
		panic(fmt.Sprintf("unknown node: %T :: %#v", node, node))
	}

	panic("unreachable")
}

// executeBlock in the given env.
// Used when entering a block, function etc.
func (i *Interpreter) executeBlock(statements []Stmt, env *Env) {
	prev := i.scope
	defer func() { i.scope = prev }()

	i.scope = env
	for _, s := range statements {
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

type funcType int

const (
	funcNone funcType = iota
	funcFunc
)

type Resolver struct {
	i      *Interpreter
	scopes []map[string]bool

	currentFunc funcType
}

func NewResolver(i *Interpreter) *Resolver {
	return &Resolver{
		i: i,
		// FIXME: Global scope?
		scopes: []map[string]bool{},

		currentFunc: funcNone,
	}
}

// execute node using this AST visitor function.
func (r *Resolver) resolve(node Node) any {
	switch v := node.(type) {
	case *BlockStmt:
		r.beginScope()
		for _, s := range v.statements {
			r.resolve(s)
		}
		r.endScope()
		return nil

	case *VarStmt:
		r.declare(v.name)
		if v.init != nil {
			r.resolve(v.init)
		}
		r.define(v.name)
		return nil

	case *Variable:
		if len(r.scopes) > 0 {
			sc := r.scopes[len(r.scopes)-1]
			if defined, ok := sc[v.name.Literal]; ok && !defined {
				runtimeErrf("Cannot read local variable in its own initializer.")
				return nil
			}
		}
		r.resolveLocal(v, v.name)
		return nil

	case *Assign:
		r.resolve(v.val)
		r.resolveLocal(v, v.name)
		return nil

	case *FuncStmt:
		r.declare(v.name)
		r.define(v.name)
		r.resolveFunction(v, funcFunc)
		return nil

	case *Grouping:
		r.resolve(v.group)
		return nil

	case *BinaryExpr:
		r.resolve(v.left)
		r.resolve(v.right)
		return nil

	case *LogicalExpr:
		r.resolve(v.left)
		r.resolve(v.right)
		return nil

	case *UnaryExpr:
		r.resolve(v.right)
		return nil

	case *Literal:
		return nil

	case *Call:
		r.resolve(v.callee)
		for _, arg := range v.args {
			r.resolve(arg)
		}
		return nil

	case *PrintStmt:
		r.resolve(v.expr)
		return nil

	case *ExprStmt:
		r.resolve(v.expr)
		return nil

	case *IfStmt:
		r.resolve(v.cond)
		r.resolve(v.thenBranch)
		if v.elseBranch != nil {
			r.resolve(v.elseBranch)
		}
		return nil

	case *WhileStmt:
		r.resolve(v.cond)
		r.resolve(v.body)
		return nil

	case *ReturnStmt:
		if r.currentFunc == funcNone {
			runtimeErrf("Can't return from top-level code")
			return nil
		}
		if v.value != nil {
			r.resolve(v.value)
		}
		return nil

	default:
		panic(fmt.Sprintf("unknown node: %T :: %#v", node, node))
	}
}

func (r *Resolver) beginScope() {
	r.scopes = append(r.scopes, map[string]bool{})
}

func (r *Resolver) endScope() {
	r.scopes = r.scopes[:len(r.scopes)-1]
}

// declare in innermost scope.
func (r *Resolver) declare(name Token) {
	if len(r.scopes) == 0 {
		return
	}
	sc := r.scopes[len(r.scopes)-1]
	if _, ok := sc[name.Literal]; ok {
		runtimeErrf("Already a variable with this name in this scope")
		return
	}

	sc[name.Literal] = false
}

// define in innermost scope.
func (r *Resolver) define(name Token) {
	if len(r.scopes) == 0 {
		return
	}
	r.scopes[len(r.scopes)-1][name.Literal] = true
}

func (r *Resolver) resolveLocal(expr Expr, name Token) {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		sc := r.scopes[i]
		if _, ok := sc[name.Literal]; ok {
			r.i.resolve(expr, len(r.scopes)-1-i)
			return
		}
	}
}

func (r *Resolver) resolveFunction(stmt *FuncStmt, kind funcType) {
	enclosing := r.currentFunc
	r.currentFunc = kind
	defer func() { r.currentFunc = enclosing }()

	r.beginScope()
	for _, p := range stmt.params {
		r.declare(p)
		r.define(p)
	}
	for _, b := range stmt.body {
		r.resolve(b)
	}
	r.endScope()
}
