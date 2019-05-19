package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"

	"gonum.org/v1/gonum/graph/encoding"
)

type node struct {
	Node
	id int64 // gonum id

	labeler NodeLabeler
}

func (n *node) ID() int64 {
	return n.id
}

func (n *node) String() string {
	return fmt.Sprintf("%v@%d", n.NodeKey(), n.id)
}

func (n *node) label() string {
	if n.labeler != nil {
		return n.labeler(n.Node)
	}
	return fmt.Sprintf("%v", n.NodeKey())
}

func (n *node) Attributes() []encoding.Attribute {
	attr := attributes{}

	if l := n.label(); l != "" {
		attr["label"] = l
	}

	return attr.Attributes()
}

type nodesOrEdges struct {
	nodes func() Nodes
	edges func() Edges
}

func (q *nodesOrEdges) Nodes() Nodes {
	return q.nodes()
}

func (q *nodesOrEdges) Edges() Edges {
	return q.edges()
}
