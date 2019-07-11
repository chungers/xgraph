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

func (edges Edges) Slice() EdgeSlice {
	all := EdgeSlice{}
	for n := range edges {
		all = append(all, n)
	}
	return all
}

func (s EdgeSlice) FromNodes() (from NodeSlice) {
	from = make(NodeSlice, len(s))
	for i := range s {
		from[i] = s[i].From()
	}
	return
}

func (s EdgeSlice) ToNodes() (to NodeSlice) {
	to = make(NodeSlice, len(s))
	for i := range s {
		to[i] = s[i].To()
	}
	return
}
