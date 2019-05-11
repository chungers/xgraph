package xgraph // import "github.com/orkestr8/xgraph"

type NodeKey interface{}
type Node interface {
	NodeKey() NodeKey
}

type Path []Node

type EdgeKind interface{}

type Edge interface {
	Kind() EdgeKind
	From() Node
	To() Node
	Context() []interface{}
}

type Options struct {
}

type GraphBuilder interface {
	Graph
	Add(Node, ...Node) error
	Associate(from Node, kind EdgeKind, to Node, optionalContext ...interface{}) (Edge, error)
}

type Nodes <-chan Node

type Graph interface {
	Has(Node) bool
	Node(NodeKey) Node
	Edge(from Node, kind EdgeKind, to Node) Edge
	To(Node, EdgeKind) Nodes
	From(Node, EdgeKind) Nodes
}
