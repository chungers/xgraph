package xgraph // import "github.com/orkestr8/xgraph"

import (
	gonum "gonum.org/v1/gonum/graph"
)

type edge struct {
	gonum   gonum.Edge
	from    Node
	to      Node
	kind    EdgeKind
	context []interface{}
}

func (e *edge) To() Node {
	return e.to
}

func (e *edge) From() Node {
	return e.from
}

func (e *edge) Context() []interface{} {
	return e.context
}

func (e *edge) Kind() EdgeKind {
	return e.kind
}
