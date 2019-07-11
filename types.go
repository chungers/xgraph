package xgraph // import "github.com/orkestr8/xgraph"

type NodeKey interface{}
type NodeKeyer interface {
	NodeKey() NodeKey
}

type Attributer interface {
	Attributes() map[string]interface{}
}

type OperatorFunc func([]interface{}) (interface{}, error)

type Operator interface {
	OperatorFunc() OperatorFunc
}

type Node interface {
	NodeKeyer
}

type Path []Node

type EdgeKind interface{}

type Edge interface {
	Attributer
	Kind() EdgeKind
	To() Node
	From() Node
}

type Options struct {

	// NodeIDOffset is the base to increment node id from.
	NodeIDOffset int64
}

type Attribute struct {
	Key   string
	Value interface{}
}

type GraphBuilder interface {
	Graph
	Add(Node, ...Node) error
	Associate(from Node, kind EdgeKind, to Node, attributes ...Attribute) (Edge, error)
}

type Nodes <-chan Node
type NodeSlice []Node

type Edges <-chan Edge
type EdgeSlice []Edge

type NodesOrEdges interface {

	// Nodes returns the nodes matching the selector. The selector is read-only and should not
	// mutate the state of the graph via associate or adding new nodes
	Nodes(...func(Node) bool) Nodes

	// Edges returns the edges matching the selector. The selector is read-only and should not
	// mutate the state of the graph via associate or adding new nodes
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
