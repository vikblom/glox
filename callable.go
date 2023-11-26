package glox

import (
	"fmt"
	"time"
)

type builtinClock struct{}

func (b *builtinClock) arity() int { return 0 }

func (b *builtinClock) call(_ *Interpreter, _ []any) any {
	return time.Now().Unix()
}

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
