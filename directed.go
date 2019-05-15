package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func newDirected(base *graph) *directed {
	return &directed{
		base:            base,
		DirectedBuilder: simple.NewDirectedGraph(),
		edges:           map[gonum.Edge]*edge{},
	}
}

type directed struct {
	base *graph
	gonum.DirectedBuilder
	edges map[gonum.Edge]*edge

	lock sync.RWMutex
}

func (d *directed) gonum(n Node, more ...Node) []gonum.Node {
	return d.base.gonum(n, more...)
}

func (d *directed) path(n gonum.Node, more ...gonum.Node) Path {
	d.lock.RLock()
	defer d.lock.RUnlock()

	all := append([]gonum.Node{n}, more...)
	out := make([]Node, len(all))

	for i, gn := range all {
		if xn, ok := gn.(*node); ok {
			out[i] = xn.Node
		}
	}
	return out
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
