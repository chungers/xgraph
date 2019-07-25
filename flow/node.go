package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"time"

	xg "github.com/orkestr8/xgraph"
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

	tasks *stopper
}

type then xg.OperatorFunc

type attributes struct {
	Timeout Duration `json:"timeout,omitempty"`
}

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
	if node.then == nil {
		node.then = func([]interface{}) (interface{}, error) {
			return nil, nil
		}
	}
	if node.tasks == nil {
		node.tasks = &stopper{}
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
	go node.gather()
	go node.scatter()

	node.tasks.waitUntil(taskGather, taskScatter)
	node.Log("Started")
	return
}

func (node *node) Close() (err error) {
	defer func() {
		e := recover()
		if e, is := e.(error); is {
			err = e
		} else {
			err = fmt.Errorf("Error closing node %v: %v", node.Node, e)
		}
		return
	}()
	if node.stop == nil {
		return
	}
	close(node.stop)

	node.tasks.waitUntilDone(taskGather, taskScatter)
	return
}

const (
	taskGather = iota
	taskScatter
)

func (node *node) gather() {
	defer func() {
		close(node.collect) // when gather completes, the other loop receiving from collect stops
		node.tasks.done(taskGather)
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

	node.tasks.add(taskGather, node.stop)
	node.Log("node.scatter", "node", node.Node)

	open := len(node.inbound) // track number of closed channels.  When all are closed, exit.
loop:
	for {
		index, value, ok := reflect.Select(cases)
		if !ok {
			open--
			if open == 0 || index == 0 { // all closed or if stop is closed
				node.Log("Exit gather", "node", node)
				return
			}
			cases[index].Chan = reflect.ValueOf(nil)
			continue loop
		}
		if value.Interface() == nil {
			panic(fmt.Errorf("Assert: value.Interface() cannot be nil."))
		}
		work, is := value.Interface().(work)
		if !is {
			panic(fmt.Errorf("Assert: value.Interface() must be instance of work: %v", value))
		}
		node.collect <- work
	}
}

func (node *node) scatter() {
	defer node.tasks.done(taskScatter)

	pending := map[flowID]gather{}

	cancel := make(chan interface{})
	node.tasks.add(taskScatter, cancel)

	node.Log("node.scatter", "node", node.Node)

loop:
	for {
		select {

		case <-cancel:
			node.Log("Exiting scatter", "node", node.Node)
			return

		case w, ok := <-node.collect:
			if !ok {
				return
			}

			node.Log("Got work", "id", w.id, "work", w)

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
				continue loop
			}

			node.Log("All input received", "id", w.id, "input", gathered, "given", node.inputFrom())

			// Build Future here
			ctx := w.ctx
			if node.attributes.Timeout > 0 {
				ctx, _ = context.WithTimeout(w.ctx, time.Duration(node.attributes.Timeout))
			}
			future := Async(ctx, func() (interface{}, error) {

				futures, err := gathered.futuresForNodes(ctx, node.inputFrom)
				if err != nil {
					return nil, err
				}
				args, err := waitFor(ctx, futures)
				if err != nil {
					return nil, err
				}

				// TODO - also pass in ctx?
				// TODO - use sync.Semaphore to set max concurrent then()?
				return node.then(args)
			})

			// Scatter / dispatch work
			node.dispatch(work{ctx: w.ctx, id: w.id, from: node.Node, Awaitable: future, callback: w.callback})

			// remove from pending list
			delete(pending, w.id)
		}
	}
}