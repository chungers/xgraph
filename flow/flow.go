package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

func NewExecutor(ref GraphRef, g xg.Graph, kind xg.EdgeKind, options Options) (Executor, error) {
	ordered, err := xg.DirectedSort(g, kind)
	if err != nil {
		return nil, err
	}
	gg, err := analyze(ref, g, kind, ordered, options)
	gg.run()
	return gg, err

}

// The buffer size is related to the number of concurrent callers we support.
// This is because at any time, a node can be accumulating partial results for
// a given caller (flowID) and becomes busy and cannot dequeue the work from
// the channel fast enough and causing clients to block for a long time.
func allocWorkChan() chan work {
	return make(chan work, 256)
}

func NewFlowGraph(g xg.Graph, kind xg.EdgeKind) (*FlowGraph, error) {
	fg := &FlowGraph{
		Graph:      g,
		Kind:       kind,
		links:      []chan work{},
		input:      map[xg.Node]chan<- work{},
		output:     map[xg.Node]chan work{},
		aggregator: allocWorkChan(),
	}
	ordered, err := xg.DirectedSort(g, kind)
	if err != nil {
		return nil, err
	}

	fg.ordered = ordered
	return fg, nil
}

func (fg *FlowGraph) outputNodes() xg.NodeSlice {
	out := xg.NodeSlice{}
	for k := range fg.output {
		out = append(out, k)
	}
	return out
}
