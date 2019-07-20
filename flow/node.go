package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	xg "github.com/orkestr8/xgraph"
)

type node struct {
	xg.Node
	Logger
	attributes attributes
	input      xg.EdgeSlice
	output     xg.EdgeSlice

	inbound  []<-chan work
	collect  chan work
	then     then
	outbound []chan<- work
	stop     chan interface{}
	tasks    sync.WaitGroup
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
	return node
}

func (node *node) dispatch(w work) {
	for _, c := range node.outbound {
		c <- w
	}
}

func (node *node) gather() {
	node.tasks.Add(1)
	defer func() {
		close(node.collect) // when gather completes, the other loop receiving from collect stops
		node.tasks.Done()
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
			panic(fmt.Errorf("value.Interface() cannot be nil."))
		}
		work, is := value.Interface().(work)
		if !is {
			panic(fmt.Errorf("value.Interface() must be instance of work: %v", value))
		}
		node.collect <- work
	}
}

func (node *node) run() {
	if node.stop == nil {
		panic(fmt.Errorf("Already stopped."))
	}
	go node.gather()
	go node.scatter()
}

func (node *node) close() {
	if node.stop == nil {
		return
	}
	close(node.stop)

	// waits for gather/scatter to complete
	node.tasks.Wait()
}

func (node *node) scatter() {
	node.tasks.Add(1)
	defer node.tasks.Done()

	pending := map[flowID]gather{}

	for {
		w, ok := <-node.collect
		if !ok {
			return
		}
		w.Log("Got work", "id", w.id, "work", w)
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

		if len(node.input) > 0 && !gathered.hasKeys(node.input.FromNodes) {
			// Nothing to do... just wait for message to come
			continue
		}

		w.Log("All input received", "id", w.id, "input", gathered, "given", node.input)

		// Build Future here
		ctx, _ := context.WithTimeout(w.ctx, time.Duration(node.attributes.Timeout))
		future := xg.Async(ctx, func() (interface{}, error) {

			futures, err := gathered.futuresForNodes(ctx, node.input.FromNodes)
			if err != nil {
				return nil, err
			}
			args, err := waitFor(ctx, futures)
			if err != nil {
				return nil, err
			}
			return node.then(args) // TODO - also pass in ctx?
		})

		// Scatter / dispatch work
		node.dispatch(work{ctx: w.ctx, id: w.id, from: node.Node, Awaitable: future, callback: w.callback})

		// remove from pending list
		delete(pending, w.id)
	}
}
