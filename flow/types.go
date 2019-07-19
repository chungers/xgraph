package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"time"

	xg "github.com/orkestr8/xgraph"
)

type Duration time.Duration

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

// graph is the executable representation.
// analyze() generates this struct. In this struct, all the channels are
// allocated and goroutines are ready to be started.
type graph struct {
	links   links
	input   xg.NodeSlice
	output  xg.NodeSlice
	ordered []*node
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

type gather map[xg.Node]xg.Awaitable
type links map[xg.Edge]chan work

type then xg.OperatorFunc

type attributes struct {
	Timeout Duration `json:"timeout,omitempty"`
}

type node struct {
	xg.Node
	attributes *attributes
	input      *input
	then       then
	output     *output
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
