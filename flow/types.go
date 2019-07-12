package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"

	xg "github.com/orkestr8/xgraph"
)

type Logger interface {
	Log(string, ...interface{})
	Warn(string, ...interface{})
}

type FlowGraph struct {
	Logger

	xg.Graph
	Kind         xg.EdgeKind
	EdgeLessFunc func(a, b xg.Edge) bool // returns True if a < b

	ordered    []xg.Node // topological sort order
	links      []chan work
	input      map[xg.Node]chan<- work
	output     map[xg.Node]chan work
	aggregator chan work
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
	Logger

	ctx      context.Context
	id       flowID
	from     xg.Node
	callback chan map[xg.Node]xg.Awaitable
}

type flowData map[xg.Node]xg.Awaitable
type links map[xg.Edge]chan work

type then xg.OperatorFunc

type node struct {
	xg.Node
	input  *input
	then   then
	output *output
}

type input struct {
	edges   xg.EdgeSlice
	recv    []<-chan work
	collect chan work
}

type output struct {
	edges xg.EdgeSlice
	send  []chan<- work
}
