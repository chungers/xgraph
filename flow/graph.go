package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"time"

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

func orderNodeByKey(a, b xg.Node) bool {
	return fmt.Sprintf("%v", a.NodeKey()) < fmt.Sprintf("%v", b.NodeKey())
}

func (g *graph) inputNodes() xg.NodeSlice {
	n := []xg.Node{}
	for k := range g.input {
		n = append(n, k)
	}
	xg.SortNodes(n, orderNodeByKey)
	return n
}

func (g *graph) outputNodes() xg.NodeSlice {
	n := []xg.Node{}
	for k := range g.output {
		n = append(n, k)
	}
	xg.SortNodes(n, orderNodeByKey)
	return n
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

func (g *graph) matchInputs(args map[xg.Node]interface{}) bool {
	test := xg.NodeSlice{}
	for k := range args {
		test = append(test, k)
	}
	xg.SortNodes(test, orderNodeByKey)
	inputs := g.inputNodes()
	if len(inputs) != len(test) {
		return false
	}
	for i, v := range inputs {
		if test[i] != v {
			return false
		}
	}
	return true
}

func (g *graph) exec(ctx context.Context, args map[xg.Node]interface{}) (<-chan gather, error) {

	if !g.matchInputs(args) {
		return nil, fmt.Errorf("incomplete input")
	}

	callback := make(chan gather)

	id := flowIDFrom(ctx)
	if id == nil {
		id = flowID(time.Now().UnixNano())
	}

	loggerFrom(ctx).Log("Start flow run", "id", id, "args", args)

	for k, v := range args {
		if ch, has := g.input[k]; has {
			source := k
			arg := v
			ch <- work{
				ctx:      ctx,
				id:       id,
				from:     source,
				callback: callback,
				Awaitable: Async(ctx, func() (interface{}, error) {
					return arg, nil
				}),
			}
		} else {
			return nil, fmt.Errorf("not an input node %v", k)
		}
	}
	return callback, nil
}
