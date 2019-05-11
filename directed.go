package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func newDirected() *directed {
	return &directed{
		DirectedBuilder: simple.NewDirectedGraph(),
		nodes:           map[int64]Node{},
		ids:             map[Node]int64{},
		edges:           map[gonum.Edge]*edge{},
	}
}

type directed struct {
	gonum.DirectedBuilder
	nodes map[int64]Node // Map of gonum node ids to xgraph nodes, which may be a subset of all xgraph nodes
	ids   map[Node]int64
	edges map[gonum.Edge]*edge

	lock sync.RWMutex
}

func (d *directed) gonum(n Node, more ...Node) []gonum.Node {
	d.lock.RLock()
	defer d.lock.RUnlock()

	all := append([]Node{n}, more...)
	out := make([]gonum.Node, len(all))
	for i, xn := range all {
		if id, has := d.ids[xn]; has {
			out[i] = d.Node(id)
		}
	}
	return out
}

func (d *directed) path(n gonum.Node, more ...gonum.Node) Path {
	d.lock.RLock()
	defer d.lock.RUnlock()

	all := append([]gonum.Node{n}, more...)
	out := make([]Node, len(all))

	for i, gn := range all {
		if xn, has := d.nodes[gn.ID()]; has {
			out[i] = xn
		}
	}
	return out
}

func (d *directed) add(n Node) gonum.Node {
	d.lock.Lock()
	defer d.lock.Unlock()

	gn := d.NewNode()
	d.AddNode(gn)
	id := gn.ID()
	d.ids[n] = id
	d.nodes[id] = n
	return gn
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
				cycles = append(cycles, dg.path(cycle[0], cycle[1:]...))
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

			sorted = dg.path(s[0], s[1:]...)
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
