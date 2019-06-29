package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"

	xg "github.com/orkestr8/xgraph"
)

func NewFlowGraphp(g xg.Graph, kind xg.EdgeKind) (*FlowGraph, error) {
	fg := &FlowGraph{
		Graph:      g,
		Kind:       kind,
		links:      []chan work{},
		input:      map[xg.Node]chan<- work{},
		output:     map[xg.Node]chan work{},
		aggregator: make(chan work),
	}
	flow, err := xg.DirectedSort(g, kind)
	if err != nil {
		return nil, err
	}

	fg.flow = flow
	return fg, nil
}

type stdout int

func (s stdout) Log(args ...interface{}) {
	if s == 0 {
		return
	}
	fmt.Println(args...)
}

type flowID int64

type work struct {
	xg.Awaitable

	ctx      context.Context
	id       flowID
	from     xg.Node
	callback chan map[xg.Node]xg.Awaitable
}

type flowData map[xg.Node]xg.Awaitable

func (m flowData) matches(gen func() []xg.Node) bool {
	matches := 0
	test := gen()
	for _, n := range test {
		_, has := m[n]
		if has {
			matches++
		}
	}
	return len(m) == len(test)
}

type edgeSlice []xg.Edge

func (s edgeSlice) from() (from []xg.Node) {
	from = make([]xg.Node, len(s))
	for i := range s {
		from[i] = s[i].From()
	}
	return
}
