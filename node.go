package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"sort"
)

type node struct {
	Node
	id int64 // gonum id

}

func (n *node) ID() int64 {
	return n.id
}

func (n *node) String() string {
	return fmt.Sprintf("%v@%d", n.NodeKey(), n.id)
}

type nodesOrEdges struct {
	nodes func([]func(Node) bool) Nodes
	edges func([]func(Edge) bool) Edges
}

func (q *nodesOrEdges) Nodes(optional ...func(Node) bool) Nodes {
	return q.nodes(optional)
}

func (q *nodesOrEdges) Edges(optional ...func(Edge) bool) Edges {
	return q.edges(optional)
}

func (nodes Nodes) Slice() NodeSlice {
	all := NodeSlice{}
	for n := range nodes {
		all = append(all, n)
	}
	return all
}

func SortNodes(nodes []Node, less func(Node, Node) bool) {
	sort.Sort(&nodeSorter{slice: nodes, less: less})
}

type nodeSorter struct {
	slice []Node
	less  func(a, b Node) bool
}

// Len is part of sort.Interface.
func (es *nodeSorter) Len() int {
	return len(es.slice)
}

// Swap is part of sort.Interface.
func (es *nodeSorter) Swap(i, j int) {
	es.slice[i], es.slice[j] = es.slice[j], es.slice[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (es *nodeSorter) Less(i, j int) bool {
	return es.less(es.slice[i], es.slice[j])
}
