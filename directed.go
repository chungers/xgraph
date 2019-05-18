package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func newDirected(base *graph, kind EdgeKind) *directed {
	return &directed{
		kind:            kind,
		nodeConverter:   base,
		DirectedBuilder: simple.NewDirectedGraph(),
		edges:           map[gonum.Edge]*edge{},
	}
}

type directed struct {
	nodeConverter
	gonum.DirectedBuilder
	edges map[gonum.Edge]*edge
	kind  EdgeKind

	lock sync.RWMutex
}

func (d *directed) associate(fromNode, toNode *node, optionalContext ...interface{}) *edge {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.Node(fromNode.id) == nil {
		d.AddNode(fromNode)
	}
	if d.Node(toNode.id) == nil {
		d.AddNode(toNode)
	}

	new := d.NewEdge(fromNode, toNode)

	ed := &edge{
		Edge:    new,
		kind:    d.kind,
		to:      toNode.Node,
		from:    fromNode.Node,
		context: optionalContext,
	}
	d.edges[new] = ed
	d.SetEdge(ed)
	return ed
}

func scopeDirected(g Graph, kind EdgeKind, do func(*directed) error) error {
	xg, ok := g.(*graph)
	if !ok {
		return ErrNotSupported{g}
	}

	directed, has := xg.directed[kind]
	if !has {
		return nil
	}

	return do(directed)
}

func DirectedCycles(g Graph, kind EdgeKind) (cycles []Path, err error) {
	err = scopeDirected(g, kind,

		func(dg *directed) error {
			cycles = []Path{}
			for _, cycle := range topo.DirectedCyclesIn(dg) {
				cycles = append(cycles, dg.xgraph(cycle[0], cycle[1:]...))
			}
			return nil
		})
	return
}

func DirectedSort(g Graph, kind EdgeKind) (sorted []Node, err error) {
	err = scopeDirected(g, kind,

		func(dg *directed) error {
			sorted = []Node{}
			s, err := topo.Sort(dg)
			if err != nil {
				return err
			}

			sorted = dg.xgraph(s[0], s[1:]...)
			return nil
		})
	return
}

func PathExistsIn(g Graph, kind EdgeKind, from, to Node) (exists bool, err error) {
	err = scopeDirected(g, kind,

		func(dg *directed) error {
			args := dg.gonum(from, to)
			exists = topo.PathExistsIn(dg, args[0], args[1])
			return nil
		})
	return
}
