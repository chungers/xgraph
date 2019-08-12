package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"io"
	"time"

	xg "github.com/orkestr8/xgraph"
)

type GraphRef string

func (ref GraphRef) NodeKey() xg.NodeKey {
	return xg.NodeKey(ref)
}

type Duration time.Duration

type Logger interface {
	Log(string, ...interface{})
	Warn(string, ...interface{})
}

type Options struct {
	Logger
}

type Executor interface {
	io.Closer
	Exec(context.Context, map[xg.Node]interface{}) (context.Context, Awaitable, error)
	ExecAwaitables(context.Context, map[xg.Node]Awaitable) (context.Context, Awaitable, error)
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

type flowID interface{}

type work struct {
	Awaitable
	Logger

	ctx      context.Context
	id       flowID
	from     xg.Node
	callback chan<- gather
}
