package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

type FlowGraph struct {
	Logger
	xg.Graph
	Kind         xg.EdgeKind
	EdgeLessFunc func(a, b xg.Edge) bool // returns True if a < b

	flow       []xg.Node // topological sort order
	links      []chan work
	input      map[xg.Node]chan<- work
	output     map[xg.Node]chan work
	aggregator chan work
}

type Logger interface {
	Log(...interface{})
}
