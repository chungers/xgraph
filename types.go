package xgraph // import "github.com/orkestr8/xgraph"

type NodeKey interface{}
type Node interface {
	NodeKey() NodeKey
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
}

type GraphBuilder interface {
	Graph
	Add(Node, ...Node) error
	Associate(from Node, kind EdgeKind, to Node, optionalContext ...interface{}) (Edge, error)
}

type Nodes <-chan Node

type Edges <-chan Edge

type NodesOrEdges interface {
	Nodes() Nodes
	Edges() Edges
}

type Graph interface {
	Node(NodeKey) Node
	Edge(from Node, kind EdgeKind, to Node) Edge
	To(Node, EdgeKind) NodesOrEdges
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
