package glox

import (
	"fmt"
	"io"
)

type callable interface {
	call(i *Interpreter, args []any) any
	arity() int
}

type LoxFunction struct {
	decl          *FuncStmt
	closure       *Env
	isInitializer bool
}

func (f *LoxFunction) arity() int {
	return len(f.decl.params)
}

func (f *LoxFunction) call(i *Interpreter, args []any) (ret any) {
	// Using panics to unwind the stack on return...
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(returnValue); ok {
				// Constructors implicitly return "this".
				if f.isInitializer {
					ret = f.closure.get("this")
					return
				}
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

	if f.isInitializer {
		return f.closure.get("this")
	}

	return nil
}

func (f *LoxFunction) String() string {
	return fmt.Sprintf("<fn %s>", f.decl.name.Literal)
}

func (f *LoxFunction) bind(inst *LoxInstance) *LoxFunction {
	env := f.closure.Fork()
	env.define("this", inst)
	return &LoxFunction{closure: env, decl: f.decl, isInitializer: f.isInitializer}
}

type LoxClass struct {
	name    string
	methods map[string]*LoxFunction
	super   *LoxClass
}

func (c *LoxClass) arity() int {
	init := c.findMethod("init")
	if init == nil {
		return 0
	}
	return init.arity()
}

// calling a class constructs an instance.
func (c *LoxClass) call(i *Interpreter, args []any) any {
	instance := &LoxInstance{class: c, fields: map[string]any{}}

	init := c.findMethod("init")
	if init != nil {
		init.bind(instance).call(i, args)
	}

	return instance
}

func (c *LoxClass) String() string {
	return fmt.Sprintf("<class %s>", c.name)
}

func (c *LoxClass) findMethod(name string) *LoxFunction {
	m, ok := c.methods[name]
	if ok {
		return m
	}
	if c.super != nil {
		return c.super.findMethod(name)
	}
	return nil
}

type LoxInstance struct {
	class  *LoxClass
	fields map[string]any
}

func (i *LoxInstance) set(name string, v any) {
	i.fields[name] = v
}

func (i *LoxInstance) get(name string) any {
	v, ok := i.fields[name]
	if ok {
		return v
	}

	m := i.class.findMethod(name)
	if m != nil {
		return m.bind(i)
	}

	runtimeErrf("Undefined property %q", name)
	return nil
}

func (i *LoxInstance) String() string {
	return fmt.Sprintf("<instance %s>", i.class.name)
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

	case *GetExpr:
		obj := i.execute(v.object)
		inst, ok := obj.(*LoxInstance)
		if !ok {
			runtimeErrf("Object %T does not have properties, must be instance.", obj)
			return nil
		}
		return inst.get(v.name.Literal)

	case *SetExpr:
		obj := i.execute(v.object)

		inst, ok := obj.(*LoxInstance)
		if !ok {
			runtimeErrf("Object %T does not have fields, must be instance.", obj)
			return nil
		}
		val := i.execute(v.value)
		inst.set(v.name.Literal, val)
		return val

	case *ThisExpr:
		return i.lookupVariable(v.keyword, v)

	case *SuperExpr:
		dist := i.locals[v]
		super, ok := i.scope.up(dist).get("super").(*LoxClass)
		if !ok {
			runtimeErrf("not a class")
			return nil
		}
		// We know the instance is just before where super is hooked on.
		obj, ok := i.scope.up(dist - 1).get("this").(*LoxInstance)
		if !ok {
			runtimeErrf("not an instance")
			return nil
		}
		method := super.findMethod(v.method.Literal)
		if method == nil {
			runtimeErrf("Undefined property %q", v.method.Literal)
		}
		return method.bind(obj)

	case *PrintStmt:
		val := i.execute(v.expr)
		fmt.Fprintf(i.out, "%v\n", val)
		return nil

	case *ExprStmt:
		_ = i.execute(v.expr)
		return nil

	case *FuncStmt:
		fn := &LoxFunction{
			decl:          v,
			closure:       i.scope,
			isInitializer: false,
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

	case *ClassStmt:
		var super *LoxClass
		if v.super != nil {
			inherited, ok := i.execute(v.super).(*LoxClass)
			if !ok {
				runtimeErrf("Superclass must be a class.")
				return nil
			}
			super = inherited
		}

		i.scope.define(v.name.Literal, nil)

		if super != nil {
			i.scope = i.scope.Fork()
			i.scope.define("super", super)
			defer func() { i.scope = i.scope.enclosing }()
		}

		methods := map[string]*LoxFunction{}
		for _, m := range v.methods {
			fun, ok := m.(*FuncStmt)
			if !ok {
				runtimeErrf("not a method")
			}
			methods[fun.name.Literal] = &LoxFunction{
				decl:          fun,
				closure:       i.scope,
				isInitializer: fun.name.Literal == "init",
			}
		}

		class := &LoxClass{
			name:    v.name.Literal,
			methods: methods,
			super:   super,
		}
		i.scope.assign(v.name.Literal, class)
		return nil

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
	funcMethod
	funcInit
)

type classType int

const (
	classNone classType = iota
	classClass
	classSub
)

type Resolver struct {
	i      *Interpreter
	scopes []map[string]bool

	currentFunc  funcType
	currentClass classType
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

	case *VarStmt:
		r.declare(v.name)
		if v.init != nil {
			r.resolve(v.init)
		}
		r.define(v.name)

	case *Variable:
		if len(r.scopes) > 0 {
			sc := r.scopes[len(r.scopes)-1]
			if defined, ok := sc[v.name.Literal]; ok && !defined {
				runtimeErrf("Cannot read local variable in its own initializer.")
				return nil
			}
		}
		r.resolveLocal(v, v.name)

	case *Assign:
		r.resolve(v.val)
		r.resolveLocal(v, v.name)

	case *FuncStmt:
		r.declare(v.name)
		r.define(v.name)
		r.resolveFunction(v, funcFunc)

	case *Grouping:
		r.resolve(v.group)

	case *BinaryExpr:
		r.resolve(v.left)
		r.resolve(v.right)

	case *LogicalExpr:
		r.resolve(v.left)
		r.resolve(v.right)

	case *UnaryExpr:
		r.resolve(v.right)

	case *Literal:

	case *Call:
		r.resolve(v.callee)
		for _, arg := range v.args {
			r.resolve(arg)
		}

	case *GetExpr:
		r.resolve(v.object)

	case *SetExpr:
		r.resolve(v.object)
		r.resolve(v.value)

	case *ThisExpr:
		if r.currentClass == classNone {
			runtimeErrf("Can't use this outside a class.")
			return nil
		}
		r.resolveLocal(v, v.keyword)

	case *SuperExpr:
		if r.currentClass == classNone {
			runtimeErrf("Can't use 'super' outside of class.")
			return nil
		}
		if r.currentClass != classSub {
			runtimeErrf("Can't use 'super' in a class with no superclass.")
			return nil
		}
		r.resolveLocal(v, v.keyword)

	case *PrintStmt:
		r.resolve(v.expr)

	case *ExprStmt:
		r.resolve(v.expr)

	case *IfStmt:
		r.resolve(v.cond)
		r.resolve(v.thenBranch)
		if v.elseBranch != nil {
			r.resolve(v.elseBranch)
		}

	case *WhileStmt:
		r.resolve(v.cond)
		r.resolve(v.body)

	case *ReturnStmt:
		if r.currentFunc == funcNone {
			runtimeErrf("Can't return from top-level code")
			return nil
		}
		if v.value != nil {
			if r.currentFunc == funcInit {
				runtimeErrf("Cannot return a value from initializer.")
				return nil
			}
			r.resolve(v.value)
		}

	case *ClassStmt:
		enclosing := r.currentClass
		r.currentClass = classClass
		defer func() { r.currentClass = enclosing }()

		r.declare(v.name)
		r.define(v.name)

		if v.super != nil {
			if v.name.Literal == v.super.name.Literal {
				runtimeErrf("A class can't inherit from itself.")
				return nil
			}
			r.currentClass = classSub // Already reset by defer.
			r.resolve(v.super)

			r.beginScope()
			r.scopes[len(r.scopes)-1]["super"] = true
		}

		r.beginScope()
		r.scopes[len(r.scopes)-1]["this"] = true

		for _, s := range v.methods {
			f := s.(*FuncStmt) // FIXME
			kind := funcMethod
			if f.name.Literal == "init" {
				kind = funcInit
			}
			r.resolveFunction(f, kind)
		}

		r.endScope()
		if v.super != nil {
			r.endScope()
		}

	default:
		panic(fmt.Sprintf("unknown node: %T :: %#v", node, node))
	}

	return nil
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
