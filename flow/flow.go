package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

func NewFlowGraph(g xg.Graph, kind xg.EdgeKind) (*FlowGraph, error) {
	fg := &FlowGraph{
		Graph:      g,
		Kind:       kind,
		links:      []chan work{},
		input:      map[xg.Node]chan<- work{},
		output:     map[xg.Node]chan work{},
		aggregator: make(chan work),
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
