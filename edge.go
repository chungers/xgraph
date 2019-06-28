package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sort"

	gonum "gonum.org/v1/gonum/graph"
)

type edge struct {
	gonum      gonum.Edge
	from       Node
	to         Node
	kind       EdgeKind
	context    []interface{}
	attributes map[string]interface{}
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
