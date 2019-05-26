package xgraph // import "github.com/orkestr8/xgraph"

type NodeKey interface{}
type NodeKeyer interface {
	NodeKey() NodeKey
}
type Node interface {
	NodeKeyer
}

type Contexter interface {
	Context() []interface{}
}

type Path []Node

type EdgeKind interface{}

type Edge interface {
	Contexter
	Kind() EdgeKind
	To() Node
	From() Node
}

type Options struct {

	// NodeIDOffset is the base to increment node id from.
	NodeIDOffset int64
}

type GraphBuilder interface {
	Graph
	Add(Node, ...Node) error
	Associate(from Node, kind EdgeKind, to Node, optionalContext ...interface{}) (Edge, error)
}

type Nodes <-chan Node

type Edges <-chan Edge

type NodesOrEdges interface {
	Nodes(...func(Node) bool) Nodes
	Edges(...func(Edge) bool) Edges
}

type Graph interface {
	Node(NodeKey) Node
	Edge(from Node, kind EdgeKind, to Node) Edge
	To(EdgeKind, Node) NodesOrEdges
	From(Node, EdgeKind) NodesOrEdges
}

type NodeShape string
type EdgeColor string

const (
	NodeShapeBox    NodeShape = "box"
	NodeShapeCircle           = "circle"
	NodeShapeOval             = "oval"
	NodeShapeRecord           = "record"

	EdgeColorBlack EdgeColor = "black"
	EdgeColorRed             = "red"
	EdgeColorBlue            = "blue"
	EdgeColorGreen           = "green"
)

type DotOptions struct {
	Name      string
	Prefix    string
	Indent    string
	NodeShape NodeShape

	Edges        map[EdgeKind]string
	EdgeColors   map[EdgeKind]EdgeColor
	EdgeLabelers map[Edge]EdgeLabeler
	NodeLabelers map[Node]NodeLabeler
}

type EdgeLabeler func(Edge) string
type NodeLabeler func(Node) string
