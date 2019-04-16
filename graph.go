package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

type node struct {
	Node
	ids map[EdgeKind]int64
}

type graph struct {
	Options
	nodes    map[Node]*node
	builders map[EdgeKind]gonum.DirectedBuilder

	lock sync.RWMutex
}

func newGraph(options Options) *graph {
	return &graph{
		Options:  options,
		nodes:    map[Node]*node{},
		builders: map[EdgeKind]gonum.DirectedBuilder{},
	}
}

func (g *graph) Add(n Node, other ...Node) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, add := range append([]Node{n}, other...) {
		found, has := g.nodes[add]
		if has && found.Node != add {
			return ErrDuplicateKey{n}
		}
	}

	for _, add := range append([]Node{n}, other...) {
		g.nodes[add] = &node{Node: add}
	}

	return nil
}

func (g *graph) Has(n Node) bool {
	g.lock.RLock()
	defer g.lock.RUnlock()

	_, has := g.nodes[n]
	return has
}

func (g *graph) Associate(kind EdgeKind, from, to Node) (Edge, error) {
	// first check for proper node membership
	if !g.Has(from) {
		return nil, ErrNoSuchNode{Node: from, context: "From"}
	}
	if !g.Has(to) {
		return nil, ErrNoSuchNode{Node: to, context: "To"}
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	// add a new graph builder if this is a new kind
	_, has := g.builders[kind]
	if !has {
		g.builders[kind] = simple.NewDirectedGraph() // TODO: copy graph, mutate, then commit at the end?
	}

	// get the node id for the Node to add edges
	if g.nodes[from].ids == nil {
		g.nodes[from].ids = map[EdgeKind]int64{}
	}
	if g.nodes[to].ids == nil {
		g.nodes[to].ids = map[EdgeKind]int64{}
	}

	get_id := func(b gonum.DirectedBuilder, _k EdgeKind, _n Node) (_id int64) {
		if _, has := g.nodes[_n].ids[_k]; !has {
			// add the node to the graph
			g.nodes[_n].ids[_k] = g.builders[_k].NewNode().ID()
		}
		return g.nodes[_n].ids[_k]
	}

	builder := g.builders[kind]
	builder.NewEdge(builder.Node(get_id(builder, kind, from)), builder.Node(get_id(builder, kind, to)))

	return nil, nil
}

func (g *graph) Edge(kind EdgeKind, from, to Node) bool {
	return false
}
