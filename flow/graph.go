package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

// graph is the executable representation.
// analyze() generates this struct. In this struct, all the channels are
// allocated and goroutines are ready to be started.
type graph struct {
	xg.Node
	input   map[xg.Node]chan<- work
	output  map[xg.Node]<-chan work
	ordered []*node
}

func (g *graph) run() {
	for _, n := range g.ordered {
		n.run()
	}
}

func (g *graph) Close() (err error) {
	for _, n := range g.ordered {
		if err = n.Close(); err != nil {
			return err
		}
	}
	return nil
}
