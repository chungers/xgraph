package xgraph // import "github.com/orkestr8/xgraph"

import (
	"strings"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
)

type edge struct {
	gonum   gonum.Edge
	from    Node
	to      Node
	kind    EdgeKind
	context []interface{}

	labeler EdgeLabeler
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

func (e *edge) label() string {
	if e.labeler != nil {
		return e.labeler(e)
	}

	labels := []string{}
	for i := range e.context {

		switch v := e.context[i].(type) {
		case func(Edge) string:
			labels = append(labels, v(e))
		case EdgeLabeler:
			labels = append(labels, v(e))
		}

	}
	return strings.Join(labels, ",")
}

func (e *edge) Attributes() []encoding.Attribute {
	attr := attributes{}
	if l := e.label(); l != "" {
		attr["label"] = l
	}
	return attr.Attributes()
}
