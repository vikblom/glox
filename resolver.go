package glox

import "fmt"

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
