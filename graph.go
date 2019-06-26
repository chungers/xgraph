package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"

	gonum "gonum.org/v1/gonum/graph"
)

type graph struct {
	Options

	nextID   *nodeID
	directed map[EdgeKind]*directed
	nodeKeys map[interface{}]*node

	lock sync.RWMutex
}

type nodeID struct {
	value int64
	lock  sync.Mutex
}

func (n *nodeID) get() (v int64) {
	n.lock.Lock()
	defer n.lock.Unlock()
	v = n.value
	n.value++
	return
}

func newGraph(options Options) *graph {
	return &graph{
		nextID:   &nodeID{value: options.NodeIDOffset},
		Options:  options,
		nodeKeys: map[interface{}]*node{},
		directed: map[EdgeKind]*directed{},
	}
}

type nodeConverter interface {
	gonum(n Node, more ...Node) []gonum.Node
	xgraph(n gonum.Node, more ...gonum.Node) []Node
}

func (g *graph) gonum(n Node, more ...Node) []gonum.Node {
	all := append([]Node{n}, more...)
	out := make([]gonum.Node, len(all))
	for i, xn := range all {
		if n, has := g.nodeKeys[xn.NodeKey()]; has {
			out[i] = n
		}
	}
	return out
}

func (g *graph) xgraph(n gonum.Node, more ...gonum.Node) []Node {
	all := append([]gonum.Node{n}, more...)
	out := make([]Node, len(all))

	for i, gn := range all {
		if xn, ok := gn.(*node); ok {
			out[i] = xn.Node
		}
	}
	return out
}

func (g *graph) setLabelers(l NodeLabeler) int {
	count := 0
	for _, n := range g.nodeKeys {
		n.labeler = l
		count++
	}
	return count
}

/*
 Add registers the given Nodes to the graph.  Duplicate key but with different identity is not allowed.
*/
func (g *graph) Add(n Node, other ...Node) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	all := append([]Node{n}, other...)
	for i := range all {
		found, has := g.nodeKeys[all[i].NodeKey()]
		if !has {
			newNode := &node{
				Node: all[i],
				id:   g.nextID.get(),
			}
			g.nodeKeys[all[i].NodeKey()] = newNode
		} else if found.Node != all[i] {
			return ErrDuplicateKey{all[i]}
		}
	}

	return nil
}

func (g *graph) Node(k NodeKey) Node {
	g.lock.RLock()
	defer g.lock.RUnlock()

	return g.nodeKeys[k]
}

func (g *graph) directedGraph(kind EdgeKind) *directed {
	g.lock.Lock()
	defer g.lock.Unlock()
	// add a new graph builder if this is a new kind
	if _, has := g.directed[kind]; !has {
		g.directed[kind] = newDirected(g, kind)
	}
	return g.directed[kind]
}

func (g *graph) Associate(from Node, kind EdgeKind, to Node, optionalContext ...interface{}) (Edge, error) {
	fromNode := g.nodeKeys[from.NodeKey()]
	if fromNode == nil {
		return nil, ErrNoSuchNode{Node: from, context: "From"}
	}
	toNode := g.nodeKeys[to.NodeKey()]
	if toNode == nil {
		return nil, ErrNoSuchNode{Node: to, context: "To"}
	}

	return g.directedGraph(kind).associate(fromNode, toNode, optionalContext...), nil
}

func (g *graph) Edge(from Node, kind EdgeKind, to Node) Edge {
	directed, has := g.directed[kind]
	if !has {
		return nil
	}

	args := directed.gonum(from, to)
	if args[0] == nil || args[1] == nil {
		return nil
	}
	return directed.edges[directed.Edge(args[0].ID(), args[1].ID())]
}

func (g *graph) From(from Node, kind EdgeKind) NodesOrEdges {
	return &nodesOrEdges{
		nodes: func(s []func(Node) bool) Nodes { return g.find(kind, from, false, s) },
		edges: func(s []func(Edge) bool) Edges { return g.findEdges(kind, from, false, s) },
	}
}

func (g *graph) To(kind EdgeKind, to Node) NodesOrEdges {
	return &nodesOrEdges{
		nodes: func(s []func(Node) bool) Nodes { return g.find(kind, to, true, s) },
		edges: func(s []func(Edge) bool) Edges { return g.findEdges(kind, to, true, s) },
	}
}

func (g *graph) find(kind EdgeKind, x Node, to bool, checks []func(Node) bool) (nodes Nodes) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	ch := make(chan Node)
	nodes = ch

	directed, has := g.directed[kind]
	if !has {
		close(ch)
		return
	}

	arg, has := g.nodeKeys[x.NodeKey()]
	if !has {
		close(ch)
		return
	}

	go func() {
		defer close(ch)

		var result gonum.Nodes
		if to {
			result = directed.To(arg.ID())
		} else {
			result = directed.From(arg.ID())
		}

	loop:
		for {
			if next := result.Next(); !next {
				break
			}

			eval := result.Node().(*node).Node

			if len(checks) == 0 {
				ch <- eval
				continue loop
			}

			for i := range checks {
				if checks[i](eval) {
					ch <- eval
					break
				}
			}
		}
	}()

	return
}

func (g *graph) findEdges(kind EdgeKind, x Node, to bool, checks []func(Edge) bool) (edges Edges) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	ch := make(chan Edge)
	edges = ch

	directed, has := g.directed[kind]
	if !has {
		close(ch)
		return
	}

	arg, has := g.nodeKeys[x.NodeKey()]
	if !has {
		close(ch)
		return
	}

	go func() {
		defer close(ch)

		var result gonum.Nodes
		if to {
			result = directed.To(arg.ID())
		} else {
			result = directed.From(arg.ID())
		}

	loop:
		for {
			if next := result.Next(); !next {
				break
			}

			var eval *edge

			if to {
				eval = directed.edges[directed.Edge(result.Node().ID(), arg.ID())]
			} else {
				eval = directed.edges[directed.Edge(arg.ID(), result.Node().ID())]
			}

			if len(checks) == 0 {
				ch <- eval
				continue loop
			}

			for i := range checks {
				if checks[i](eval) {
					ch <- eval
					break
				}
			}
		}
	}()

	return
}
