package xgraph // import "github.com/orkestr8/xgraph"

import (
	gonum "gonum.org/v1/gonum/graph"
)

type edge struct {
	gonum      gonum.Edge
	from       Node
	to         Node
	kind       EdgeKind
	attributes []Attribute
}

func (e *edge) To() Node {
	return e.to
}

func (e *edge) From() Node {
	return e.from
}

func (e *edge) Kind() EdgeKind {
	return e.kind
}

func (e *edge) Attributes() map[string]interface{} {
	m := map[string]interface{}{}
	for _, a := range e.attributes {
		m[a.Key] = a.Value
	}
	return m
}
