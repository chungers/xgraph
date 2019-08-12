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

func (g *graph) checkComplete(args map[xg.Node]Awaitable) bool {
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

func (g *graph) execFutures(ctx context.Context, args map[xg.Node]Awaitable) (context.Context, <-chan gather, error) {

	id := flowIDFrom(ctx)
	if id == nil {
		id = flowID(time.Now().UnixNano())
		ctx = setFlowID(ctx, id) // set the id in the context if not already set
	}

	callback := gatherChanFrom(ctx)
	if callback == nil {
		callback = make(chan gather)
		ctx = setGatherChan(ctx, callback)
	}

	loggerFrom(ctx).Log("Start flow run", "id", id, "args", args)

	for k, awaitable := range args {
		if ch, has := g.input[k]; has {
			source := k
			ch <- work{
				ctx:       ctx,
				id:        id,
				from:      source,
				callback:  callback,
				Awaitable: awaitable,
			}
		} else {
			return ctx, nil, fmt.Errorf("not an input node %v", k)
		}
	}

	if !g.checkComplete(args) {
		// if this set of args is incomplete,
		// set up a watch for timeout
		go func() {
			defer tryClose(loggerFrom(ctx), callback)

			<-ctx.Done() // canceled or timedout
			ret := map[xg.Node]Awaitable{}
			for _, k := range g.outputNodes() {
				ret[k] = Const(ctx.Err())
			}
			callback <- ret
		}()
	}
	return ctx, callback, nil
}

func (g *graph) execValues(ctx context.Context, args map[xg.Node]interface{}) (context.Context, <-chan gather, error) {
	awaitables := map[xg.Node]Awaitable{}
	for k, v := range args {
		node := k
		awaitables[node] = Const(v)
	}
	return g.execFutures(ctx, awaitables)
}

func (g *graph) Exec(ctx context.Context, args map[xg.Node]interface{}) (context.Context, Awaitable, error) {

	ctx, ch, err := g.execValues(ctx, args)
	if err != nil {
		return ctx, nil, err
	}

	aw := awaitableFrom(ctx)
	if aw == nil {
		aw = Async(ctx, func() (interface{}, error) {
			return map[xg.Node]Awaitable(<-ch), nil
		})
		ctx = setAwaitable(ctx, aw)
	}

	return ctx, aw, err
}

func (g *graph) ExecAwaitables(ctx context.Context, args map[xg.Node]Awaitable) (context.Context, Awaitable, error) {

	ctx, ch, err := g.execFutures(ctx, args)
	if err != nil {
		return ctx, nil, err
	}

	aw := awaitableFrom(ctx)
	if aw == nil {
		aw = Async(ctx, func() (interface{}, error) {
			return map[xg.Node]Awaitable(<-ch), nil
		})
		ctx = setAwaitable(ctx, aw)
	}

	return ctx, aw, err
}
