package glox

// TODO: Expr.String().
type Expr interface {
	// TODO
	express()
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

func (e *BinaryExpr) express() {

}

func (e *UnaryExpr) express() {

}

func (e *Literal) express() {

}

func (e *Grouping) express() {

}
