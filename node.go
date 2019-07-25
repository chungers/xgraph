package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
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
