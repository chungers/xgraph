package xgraph // import "github.com/orkestr8/xgraph"

type NodeKey []byte
type Node interface {
	NodeKey() NodeKey
}

type Path []Node

type EdgeKind interface{}

type Edge interface {
	Kind() EdgeKind
	From() Node
	To() Node
}

type Options struct {
}

type Graph interface {
	Add(Node, ...Node) error
	Associate(from Node, kind EdgeKind, to Node) (Edge, error)
	Has(Node) bool
	Node(NodeKey) Node
	Edge(from Node, kind EdgeKind, to Node) bool
}
