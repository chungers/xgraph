package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sort"
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

	labeler EdgeLabeler
}

func (e *edge) label() string {
	if e.labeler != nil {
		return e.labeler(&edgeView{e})
	}

	labels := []string{}
	for i := range e.context {

		switch v := e.context[i].(type) {
		case func(Edge) string:
			labels = append(labels, v(&edgeView{e}))
		case EdgeLabeler:
			labels = append(labels, v(&edgeView{e}))
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
	return e.context
}

func (e *edgeView) Kind() EdgeKind {
	return e.kind
}

func SortEdges(edges []Edge, less func(Edge, Edge) bool) {
	sort.Sort(&edgeSorter{slice: edges, less: less})
}

type edgeSorter struct {
	slice []Edge
	less  func(a, b Edge) bool
}

// Len is part of sort.Interface.
func (es *edgeSorter) Len() int {
	return len(es.slice)
}

// Swap is part of sort.Interface.
func (es *edgeSorter) Swap(i, j int) {
	es.slice[i], es.slice[j] = es.slice[j], es.slice[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (es *edgeSorter) Less(i, j int) bool {
	return es.less(es.slice[i], es.slice[j])
}
