package xgraph // import "github.com/orkestr8/xgraph"

type NodeKey []byte
type Node interface {
	Key() NodeKey
}

type EdgeKind int

type Edge interface {
	Kind() EdgeKind
	From() Node
	To() Node
	Reverse() Edge
}

type Options struct {
}

type Graph interface {
	Add(Node, ...Node) error
	Associate(kind EdgeKind, from, to Node) (Edge, error)
	Has(Node) bool
	Edge(kind EdgeKind, from, to Node) bool
}
