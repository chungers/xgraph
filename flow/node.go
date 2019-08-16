package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"time"

	xg "github.com/orkestr8/xgraph"
	"golang.org/x/sync/semaphore"
)

type node struct {
	io.Closer
	xg.Node
	Logger
	attributes attributes
	inputFrom  func() xg.NodeSlice
	outputTo   func() xg.NodeSlice

	inbound  []<-chan work
	collect  chan work
	then     then
	outbound []chan<- work
	stop     chan interface{}

	terminal bool
	sem      *semaphore.Weighted
}

type then xg.OperatorFunc

// sets default values for the receiver
func (node *node) defaults() *node {
	if node.Logger == nil {
		node.Logger = logger(0)
	}
	if node.Node == nil {
		panic(fmt.Errorf("Missing Node: %v", node))
	}
	if &node.attributes == nil {
		node.attributes = attributes{}
	}
	if node.collect == nil {
		node.collect = make(chan work)
	}
	if node.stop == nil {
		node.stop = make(chan interface{})
	}
	return node
}

func (node *node) dispatch(w work) {
	for _, c := range node.outbound {
		c <- w
	}
}

// run() blocks until gather and scatter are all running to avoid races.
func (node *node) run() {
	if node.stop == nil {
		panic(fmt.Errorf("node.stop == nil. run() is not idempotent."))
	}

	gC := make(chan interface{})
	sC := make(chan interface{})
	go node.gather(gC)
	go node.scatter(sC)
	<-gC
	<-sC
	node.Log("Started", "node", node.Node)
	return
}

func (node *node) Close() (err error) {
	defer func() {
		e := recover()
		if e, is := e.(error); is {
			err = e
		}
		return
	}()
	if node.stop == nil {
		return
	}
	close(node.stop)

	return
}

func (node *node) gather(ready chan interface{}) {
	defer func() {
		close(node.collect) // when gather completes, the other loop receiving from collect stops
	}()

	if node.stop == nil {
		panic(fmt.Errorf("gather: node.stop cannot be nil"))
	}

	cases := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(node.stop),
		},
	}
	for i := range node.inbound {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(node.inbound[i]),
		})
	}

	close(ready)
	node.Log("node.gather", "node", node.Node)

	open := len(node.inbound) // track number of closed channels.  When all are closed, exit.
loop:
	for {
		index, value, ok := reflect.Select(cases)
		if !ok {
			open--
			if open == 0 || index == 0 { // all closed or if stop is closed
				node.Log("exit node.gather", "node", node)
				return
			}
			cases[index].Chan = reflect.ValueOf(nil)
			continue loop
		}
		if value.Interface() == nil {
			panic(fmt.Errorf("assert: value.Interface() cannot be nil."))
		}
		work, is := value.Interface().(work)
		if !is {
			panic(fmt.Errorf("assert: value.Interface() must be instance of work: %v", value))
		}
		node.collect <- work
	}
}

func tryClose(logger Logger, c chan<- gather) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warn("recovered:", e)
		}
	}()
	close(c)
}

func (node *node) scatter(ready chan interface{}) {
	pending := map[flowID]gather{}

	close(ready)

	node.Log("node.scatter", "node", node.Node)

	for {
		w, ok := <-node.collect
		if !ok {
			node.Log("Exiting scatter", "node", node.Node)
			return
		}

		// match messages by flow id.
		gathered, has := pending[w.id]
		if !has {
			gathered = gather{}
			pending[w.id] = gathered
		}

		if prev, has := gathered[w.from]; has {
			// Warning that old value will be replaced by duplicate/new
			w.Warn("Duplicate awaitable", "id", w.id,
				"from", w.from, "old", prev, "new", w)
		}
		gathered[w.from] = w

		if len(node.inputFrom()) > 0 && !gathered.hasKeys(node.inputFrom) {
			// Nothing to do... just wait for message to come
			continue
		}

		node.Log("All input received", "node", node.Node, "id", w.id, "input", gathered)

		// remove from pending list
		delete(pending, w.id)

		if w.callback != nil && node.terminal {
			// Send the gathered futures to callback without blocking
			select {
			case w.callback <- gathered:
				tryClose(w.Logger, w.callback)
			default:
			}
			continue
		}

		// Build Future to pass on to the next stages
		ctx := w.ctx
		if node.attributes.Timeout > 0 {
			ctx, _ = context.WithTimeout(w.ctx, time.Duration(node.attributes.Timeout))
		}

		// Test for the case where this is the graph input node and the gathered input is
		// data for itself.  This is for the case when a graph's input node is wired to
		// have the collect channel receiving work from outside the graph.  See graph.exec().
		var future Awaitable
		switch {
		case len(gathered) == 1 && gathered[node.Node] != nil:
			future = gathered[node.Node] // just use the future given in the work message

		case node.attributes.Inline:
			// TODO - This case will hang in TestFlowExecFull
			future = node.applyInline(ctx, gathered) // inputs from other nodes and not itself

		default:
			future = node.applyAsync(ctx, gathered) // inputs from other nodes and not itself
		}

		// Scatter / dispatch work
		node.dispatch(work{ctx: w.ctx, id: w.id, from: node.Node, Awaitable: future, callback: w.callback})
	}

}

func (node *node) applyInline(ctx context.Context, m gather) Awaitable {

	return Inline(func() (interface{}, error) {

		keys, futures, err := m.futuresForNodes(ctx, node.inputFrom)
		if err != nil {
			return nil, err
		}

		// futures and inputFrom are 1:1, so args and inputFrom are 1:1
		args, err := waitFor(futures)
		if err != nil {
			return nil, err
		}

		if node.then == nil {
			// If no operator, simply return a map of all the values by Node.
			// This is used for a special collection node for the entire graph.
			m := map[xg.Node]interface{}{}
			for i := range keys {
				m[keys[i]] = args[i]
			}
			return m, nil
		}

		// Do work rather than collect the data.
		return node.apply(ctx, args)
	})
}

func (node *node) applyAsync(ctx context.Context, m gather) Awaitable {

	return Async(ctx, func() (interface{}, error) {

		keys, futures, err := m.futuresForNodes(ctx, node.inputFrom)
		if err != nil {
			return nil, err
		}

		// futures and inputFrom are 1:1, so args and inputFrom are 1:1
		args, err := waitFor(futures)
		if err != nil {
			return nil, err
		}

		if node.then == nil {
			// If no operator, simply return a map of all the values by Node.
			// This is used for a special collection node for the entire graph.
			m := map[xg.Node]interface{}{}
			for i := range keys {
				m[keys[i]] = args[i]
			}
			return m, nil
		}

		// Do work rather than collect the data.
		return node.apply(ctx, args)
	})
}

func (node *node) apply(ctx context.Context, args []interface{}) (interface{}, error) {
	// Using semaphore. This allows us to do throttling or limit
	// the number of parallel workers.
	if node.sem != nil {
		err := node.sem.Acquire(ctx, 1)
		if err != nil {
			return nil, err
		}
		defer node.sem.Release(1)
	}
	return node.then(args)
}
