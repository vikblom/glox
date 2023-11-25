package glox

import "time"

type builtinClock struct{}

func (b *builtinClock) arity() int { return 0 }

func (b *builtinClock) call(_ *Interpreter, _ []any) any {
	return time.Now().Unix()
}
