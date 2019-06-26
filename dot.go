package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"strconv"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
)

// dotNode is a view of the Node.  It implements dot related methods
// and for labeling purposes.  It also implements the Setter interfaces
// called when decoding a dotfile.
//
// This implementation of Node interface is provided because the typical
// usage of the xgraph API is for user to provide their own types that
// implement the Node interface.  When parsing dotfile, this type is
// used as the default implementation.
type dotNode struct {
	key        NodeKey
	id         int64
	attributes map[string]string
}

func (n *dotNode) SetDOTID(id string) {
	n.key = NodeKey(id)
}

func (n dotNode) ID() int64 {
	return int64(n.id)
}

func (n dotNode) NodeKey() NodeKey {
	return n.key
}

func (n dotNode) SetAttribute(attr encoding.Attribute) error {
	if n.attributes == nil {
		n.attributes = map[string]string{}
	}
	n.attributes[attr.Key] = attr.Value
	return nil
}

type dotSubgraph struct {
	*directed
	DotOptions
	dot.Subgrapher
}

func (ds *dotSubgraph) DOTID() string {
	dotid := fmt.Sprintf("%v", ds.kind)
	if ds.DotOptions.Edges != nil {
		if v, has := ds.DotOptions.Edges[ds.kind]; has {
			dotid = v
		}
		return dotid
	}
	return dotid
}

func (dg *dotSubgraph) edgeColor() string {
	c := "black"
	if dg.DotOptions.EdgeColors != nil {
		if v, has := dg.DotOptions.EdgeColors[dg.kind]; has {
			c = string(v)
		}
	}
	return c
}

func (dg *dotSubgraph) edgeLabel() string {
	return dg.DOTID()
}

func (dg *dotSubgraph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	graph = attributes{}
	node = attributes{}
	edge = attributes{"color": dg.edgeColor(), "label": dg.edgeLabel()}
	return
}

func (ds *dotSubgraph) Subgraph() gonum.Graph {
	return ds.directed
}

type dotGraph struct {
	DotOptions
	gonum.Directed

	xg *graph
}

func (dg *dotGraph) Node(id int64) gonum.Node {
	return dg.Directed.Node(id)
}

func (dg *dotGraph) DOTID() string {
	id := dg.Name
	if id == "" {
		return "G"
	}
	return id
}

type attributes map[string]string

func (a attributes) Attributes() []encoding.Attribute {
	out := []encoding.Attribute{}
	for k, v := range a {
		out = append(out, encoding.Attribute{Key: k, Value: v})
	}
	return out
}

func (dg *dotGraph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	graph = attributes{}
	node = attributes{"shape": string(dg.DotOptions.NodeShape)}
	edge = attributes{}
	return
}

func (dg *dotGraph) Structure() []dot.Graph {
	subs := []dot.Graph{}

	for k := range dg.xg.directed {
		sg := &dotSubgraph{
			directed:   dg.xg.directed[k],
			DotOptions: dg.DotOptions,
		}
		subs = append(subs, sg)
	}
	return subs
}

func EncodeDot(g Graph, options DotOptions) ([]byte, error) {

	xg, is := g.(*graph)
	if !is {
		return nil, ErrNotSupported{g}
	}

	// TODO - replace all this setting of labels with
	// a wrapper dotNode that has the labeler.

	// Set any EdgeLabelers to customize labels for Edges
	count := 0
	for edg, labeler := range options.EdgeLabelers {
		if ev, is := edg.(*edge /*View*/); is {
			ev.labeler = labeler
			count++
		}
	}
	if count > 0 {
		defer func() {
			// reset the labeler so we don't interfere with future renders
			for edg := range options.EdgeLabelers {
				if ev, is := edg.(*edge /*View*/); is {
					ev.labeler = nil
				}
			}
		}()
	}

	// Set any NodeLabelers to customize labels for Nodes
	count = 0

	// Check if a global labeler is set
	if labeler, has := options.NodeLabelers[nil]; has {
		count = xg.setLabelers(labeler)
	}
	for n, labeler := range options.NodeLabelers {
		if n != nil {
			if v := g.Node(n.NodeKey()); v != nil {
				if xn, is := v.(*node); is {
					xn.labeler = labeler
					count++
				}
			}
		}
	}
	if count > 0 {
		defer func() {
			xg.setLabelers(nil)
		}()
	}

	dg := &dotGraph{
		DotOptions: options,
		Directed:   simple.NewDirectedGraph(), //xg.DirectedBuilder,
		xg:         xg,
	}

	return dot.Marshal(dg, options.Name, options.Prefix, options.Indent)
}

func DecodeDot(buff []byte, g Graph, kind EdgeKind) error {
	// Check the implementation. Currently only support our own.
	xg, is := g.(*graph)
	if !is {
		return fmt.Errorf("wrong implementation")
	}

	directed := xg.directedGraph(kind)
	return dot.Unmarshal(buff, &dotBuilder{Graph: directed, xg: xg, kind: kind})
}

type dotBuilder struct {
	gonum.Graph
	kind EdgeKind
	xg   *graph
}

func (b *dotBuilder) NewNode() gonum.Node {
	return &dotNode{id: b.xg.nextID.get()}
}

func (b *dotBuilder) AddNode(n gonum.Node) {
	add, is := n.(*dotNode)
	if !is {
		return
	}
	b.xg.Add(add)
	return
}

type dotEdge struct {
	edge       *edge
	from       gonum.Node
	to         gonum.Node
	attributes map[string]string
}

func (e dotEdge) From() gonum.Node {
	return e.from
}

func (e dotEdge) To() gonum.Node {
	return e.to
}

func (e dotEdge) ReversedEdge() gonum.Edge {
	return &dotEdge{to: e.from, from: e.to}
}

func (e *dotEdge) SetAttribute(attr encoding.Attribute) error {
	if e.attributes == nil {
		e.attributes = map[string]string{}
	}
	e.attributes[attr.Key] = attr.Value
	if attr.Key == "context" {
		// TODO - hacky
		if intV, err := strconv.Atoi(attr.Value); err == nil {
			e.edge.context = []interface{}{intV}
		}
	}
	return nil
}

func (b *dotBuilder) NewEdge(from, to gonum.Node) gonum.Edge {
	return &dotEdge{from: from, to: to}
}

func (b *dotBuilder) SetEdge(e gonum.Edge) {
	ee, is := e.(*dotEdge)
	if !is {
		return
	}
	from, is := e.From().(*dotNode)
	if !is {
		return
	}
	to, is := e.To().(*dotNode)
	if !is {
		return
	}
	add, err := b.xg.Associate(from, b.kind, to)
	if err == nil {
		xedge, is := add.(*edge)
		if is {
			ee.edge = xedge
		}
	}
	return
}

func (b *dotBuilder) DOTAttributeSetters() (G, N, E encoding.AttributeSetter) {
	fmt.Println("DOTAttributeSetters")
	return attr("G"), attr("N"), attr("E")
}

type attr string

func (a attr) SetAttribute(attr encoding.Attribute) error {
	fmt.Println(a, "SetAttribute", attr.Key, attr.Value)
	return nil
}
