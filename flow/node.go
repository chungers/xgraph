package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"reflect"
	"time"

	xg "github.com/orkestr8/xgraph"
)

type node struct {
	xg.Node
	Logger
	attributes *attributes
	input      xg.EdgeSlice
	inbound    []<-chan work
	collect    chan work
	then       then
	output     xg.EdgeSlice
	outbound   []chan<- work
	stop       chan interface{}
}

type then xg.OperatorFunc

type attributes struct {
	Timeout Duration `json:"timeout,omitempty"`
}

func (node *node) dispatch(w work) {
	for _, c := range node.outbound {
		c <- w
	}
}

func (node *node) gather() {

	defer close(node.collect) // when gather completes, the other loop receiving from collect stops

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
				return
			}
			cases[index].Chan = reflect.ValueOf(nil)
			continue loop
		}
		if value.Interface() == nil {
			continue loop
		}
		work, is := value.Interface().(work)
		if !is {
			continue loop
		}
		node.collect <- work
	}
}

func (node *node) run() {
	go node.gather()
	go node.scatter()
}

func (node *node) close() {
	if node.stop == nil {
		return
	}
	close(node.stop)
	node.stop = nil
}

func (node *node) scatter() {

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
