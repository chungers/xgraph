package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"strings"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
)

type edge struct {
	gonum.Edge
	from    Node
	to      Node
	kind    EdgeKind
	context []interface{}
}

func (e *edge) label() string {
	if len(e.context) > 0 {
		s := make([]string, len(e.context))
		for i := range e.context {
			s[i] = fmt.Sprintf("%v", e.context[i])
		}
		return strings.Join(s, ",")
	}
	return ""
}

func (e *edge) Attributes() []encoding.Attribute {
	attr := attributes{}
	if l := e.label(); l != "" {
		attr["label"] = l
	}
	return attr.Attributes()
}

// edgeView is used to work around the problem that gonum.Edge.From() and xgraph.Edge.From()
// cant be disambiguated by the compiler (different return types).  We want to separate the
// api implementation from the low-level implementations like dot.Attributer as well.
type edgeView struct {
	*edge
}

func (e *edgeView) To() Node {
	return e.to
}

func (e *edgeView) From() Node {
	return e.from
}

func (e *edgeView) Context() []interface{} {
	return e.edge.context
}

func (e *edgeView) Kind() EdgeKind {
	return e.kind
}
