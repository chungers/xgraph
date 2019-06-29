package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"strings"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
)

type attributes map[string]string

func (a attributes) Attributes() []encoding.Attribute {
	out := []encoding.Attribute{}
	for k, v := range a {
		out = append(out, encoding.Attribute{Key: k, Value: v})
	}
	return out
}

// For encoding to dotfile, dotNode is a view of the Node.
// It implements dot related methods and labeling.
// For decoding it also implements the Setter interfaces called when
// decoding a dotfile.
//
// During decoding this implementation of Node interface is provided
// because the typical usage of the xgraph API is for user to provide
// their own types that implement the Node interface.
// When parsing dotfile, this type is used as the default implementation.
type dotNode struct {
	key        NodeKey
	id         int64
	attributes map[string]string
	attributer Attributer
	labeler    NodeLabeler
}

func (n dotNode) DOTID() string {
	return fmt.Sprintf("%v", n.key)
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

func (n *dotNode) SetAttribute(attr encoding.Attribute) error {
	if n.attributes == nil {
		n.attributes = map[string]string{}
	}
	n.attributes[attr.Key] = attr.Value
	return nil
}

func (n dotNode) label() string {
	if n.labeler != nil {
		return n.labeler(nil)
	}
	return fmt.Sprintf("%v", n.NodeKey())
}

func (n dotNode) Attributes() []encoding.Attribute {
	attr := attributes{}

	// Merge all attributes
	// Note that Attributer interface is for the case when
	// user provides Node implementation and satisfies the
	// Attributer interface contract.  The dotNode.attributes
	// field is used when dotNode itself is used as the user
	// provided Node implementation (from parsing the dotfile)
	if n.attributer != nil {
		for k, v := range n.attributer.Attributes() {
			attr[k] = fmt.Sprintf("%v", v)
		}
	}

	for k, v := range n.attributes {
		attr[k] = v
	}
	if l := n.label(); l != "" {
		attr["label"] = l
	}
	return attr.Attributes()
}

type dotNodes struct {
	gonum.Nodes
	dg *dotGraph
}

func (dn *dotNodes) Node() gonum.Node {
	gn := dn.Nodes.Node()
	return dn.dg.dotNode(gn)
}

func (dg *dotGraph) Nodes() gonum.Nodes {
	return &dotNodes{
		Nodes: dg.Directed.Nodes(),
		dg:    dg,
	}
}

func (dg *dotGraph) From(id int64) gonum.Nodes {
	return &dotNodes{
		Nodes: dg.Directed.From(id),
		dg:    dg,
	}
}

func (dg *dotGraph) Edge(uid, vid int64) gonum.Edge {
	return dg.dotEdge(dg.Directed.Edge(uid, vid))
}

type dotGraph struct {
	DotOptions
	gonum.Directed

	kind EdgeKind // set only when it's a subgraph
	xg   *graph
}

func (dg *dotGraph) dotNode(gn gonum.Node) gonum.Node {
	if gn == nil {
		return nil
	}

	v := dg.xg.xgraph(gn)
	if len(v) == 0 {
		return nil
	}

	labeler, has := dg.DotOptions.NodeLabelers[v[0]]
	if has {
		old := labeler
		userV := v[0] // this is actually the pointer to user provided node
		labeler = func(Node) string {
			return old(userV)
		}
	}
	// Special case when we use dotfile as input where we provided our own
	// Node implementation struct (dotNode): don't wrap it again.
	if dn, is := v[0].(*dotNode); is {
		dn.labeler = labeler
		return dn
	}
	return &dotNode{
		key:     v[0].NodeKey(),
		id:      gn.ID(),
		labeler: labeler,
	}
}

func (dg *dotGraph) dotEdge(e gonum.Edge) gonum.Edge {
	if de, is := e.(*dotEdge); is {
		return de
	}
	xedge, has := dg.xg.directed[dg.kind].edges[e]
	if !has {
		return e
	}

	return &dotEdge{
		edge: xedge,
		from: e.From(),
		to:   e.To(),
	}
}

func (dg *dotGraph) Node(id int64) gonum.Node {
	gn := dg.Directed.Node(id)
	return dg.dotNode(gn)
}

func (dg *dotGraph) DOTID() string {
	if dg.kind == nil {
		id := dg.Name
		if id == "" {
			return "G"
		}
		return id
	}

	dotid := fmt.Sprintf("%v", dg.kind)
	if dg.DotOptions.Edges != nil {
		if v, has := dg.DotOptions.Edges[dg.kind]; has {
			dotid = v
		}
		return dotid
	}
	return dotid
}

func (dg dotGraph) edgeLabel() string {
	return dg.DOTID()
}

func (dg dotGraph) edgeColor() string {
	c := "black"
	if dg.DotOptions.EdgeColors != nil {
		if v, has := dg.DotOptions.EdgeColors[dg.kind]; has {
			c = string(v)
		}
	}
	return c
}

func (dg dotGraph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	graph = attributes{}
	node = attributes{"shape": string(dg.DotOptions.NodeShape)}
	edge = attributes{"color": dg.edgeColor(), "label": dg.edgeLabel()}
	return
}

func (dg *dotGraph) Structure() []dot.Graph {
	if dg.kind != nil {
		return nil
	}

	subs := []dot.Graph{}
	for k := range dg.xg.directed {
		subs = append(subs,
			&dotGraph{
				kind:       k,
				DotOptions: dg.DotOptions,
				Directed:   dg.xg.directed[k],
				xg:         dg.xg,
			})
	}
	return subs
}

func EncodeDot(g Graph, options DotOptions) ([]byte, error) {
	xg, is := g.(*graph)
	if !is {
		return nil, ErrNotSupported{g}
	}
	dg := &dotGraph{
		DotOptions: options,
		Directed:   simple.NewDirectedGraph(),
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
	return dot.Unmarshal(buff,
		&dotBuilder{
			Graph:  directed,
			xg:     xg,
			kind:   kind,
			nextID: &nodeID{},
		})
}

type dotBuilder struct {
	gonum.Graph
	kind   EdgeKind
	xg     *graph
	nextID *nodeID
}

func (b *dotBuilder) NewNode() gonum.Node {
	return &dotNode{id: b.nextID.get()}
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
	edge    *edge
	from    gonum.Node
	to      gonum.Node
	labeler EdgeLabeler
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
	if e.edge == nil {
		panic("cna't set")
	}
	if e.edge.attributes == nil {
		e.edge.attributes = []Attribute{}
	}
	e.edge.attributes = append(e.edge.attributes,
		Attribute{Key: attr.Key, Value: attr.Value})
	return nil
}

func (e dotEdge) label() string {
	if e.labeler != nil {
		return e.labeler(e.edge)
	}

	if e.edge.attributes == nil {
		return ""
	}
	for _, a := range e.edge.attributes {
		if a.Key == "label" {
			return fmt.Sprintf("%v", a.Value)
		}
	}

	labels := []string{}
	for _, a := range e.edge.attributes {
		switch v := a.Value.(type) {
		case func(Edge) string:
			labels = append(labels, v(e.edge))
		case EdgeLabeler:
			labels = append(labels, v(e.edge))
		}

	}
	return strings.Join(labels, ",")
}

func (e dotEdge) Attributes() []encoding.Attribute {
	if e.edge == nil {
		return nil
	}
	attr := attributes{}

	for k, v := range e.edge.Attributes() {
		attr[k] = fmt.Sprintf("%v", v)
	}

	if l := e.label(); l != "" {
		attr["label"] = l
	}
	return attr.Attributes()
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
	return attr("G"), attr("N"), attr("E")
}

type attr string

func (a attr) SetAttribute(attr encoding.Attribute) error {
	return nil
}
