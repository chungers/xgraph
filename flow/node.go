package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"time"

	xg "github.com/orkestr8/xgraph"
)

type then xg.OperatorFunc

type input struct {
	node  xg.Node
	edges xg.EdgeSlice
	ch    chan work
}

type output struct {
	then  then
	edges xg.EdgeSlice
	input *input
}

func gather(in xg.EdgeSlice, this xg.Node) *input {
	return &input{edges: in, ch: make(chan work), node: this}
}

func (in *input) then(then then) *output {
	return &output{input: in, then: then}
}

func (out *output) scatter(output xg.EdgeSlice) func() {
	out.edges = output
	return func() {
		eventLoop(out.input.edges, out.input.ch, out.then, out, out.input.node)
	}
}

func eventLoop(input xg.EdgeSlice, collector <-chan work, then then, output *output, this xg.Node) {

	pending := map[flowID]flowData{}

	for {
		w, ok := <-collector
		if !ok {
			return
		}
		w.Log("Got work", "id", w.id, "work", w)
		// match messages by flow id.
		fd, has := pending[w.id]
		if !has {
			fd = flowData{}
			pending[w.id] = fd
		}
		if prev, has := fd[w.from]; has {
			// Warning that old value will be replaced by duplicate/new
			w.Warn("Duplicate awaitable", "id", w.id,
				"from", w.from, "old", prev, "new", w)
		}
		fd[w.from] = w

		if len(input) > 0 && !fd.hasKeys(xg.EdgeSlice(input).FromNodes) {
			// Nothing to do... just wait for message to come
			continue
		}

		w.Log("All input received", "id", w.id, "input", fd, "given", input)

		// Build Future here
		ctx, _ := context.WithTimeout(w.ctx, 1*time.Second)
		future := xg.Async(ctx, func() (interface{}, error) {
			args, err := fd.args(ctx, input)
			if err != nil {
				return nil, err
			}
			return then(args) // TODO - also pass in ctx?
		})

		// Scatter / dispatch work
		next := work{ctx: w.ctx, id: w.id, from: this, Awaitable: future, callback: w.callback}
		if next.ctx != nil {
		}
		// remove from pending list
		delete(pending, w.id)
	}
}

/*

loop_input:
	for {
		w, ok := <-collector
		if !ok {
			return
		}
		w.Log("Got work", "id", w.id, "work", w)
		// match messages by flow id.
		inputMap, has := pending[w.id]
		if !has {
			inputMap = flowData{}
			pending[w.id] = inputMap
		}
		if prev, has := inputMap[w.from]; has {
			// Warning that old value will be replaced by duplicate/new
			w.Warn("Duplicate awaitable", "id", w.id,
				"from", w.from, "old", prev, "new", w)
		}
		inputMap[w.from] = w

		if len(to) > 0 && !inputMap.matches(edgeSlice(to).from) {
			// Nothing to do... just wait for message to come
			continue loop_input
		}

		w.Log("All input received", "id", w.id,
			"input", inputMap, "given", to)

		// Now inputs are collected.  Build another future and pass it on.
		// TODO - context with timeout
		if w.ctx == nil {
			panic("nil ctx -- coding error. Must pass in context.")
		}

		ctx := w.ctx
		received := w

		future := xg.Async(ctx, func() (interface{}, error) {

			if len(to) == 0 {
			}

			args := []interface{}{}
			if len(to) > 0 {
				// Wait for all inputs to complete computation and build args
				// for this node before proceeding with this node's computation.
				for i := range to {
					future := inputMap[to[i].From()]

					if future == nil {
						panic(fmt.Errorf("%v : Missing future for %v", this, to[i]))
					}
					// Calling the Value or Error will block until completion
					// TODO - a stuck future will lock this entirely. Add deadline.
					if err := future.Error(); err != nil {
						fg.Log("Error received", "id", w.id, "op", w, "err", err)
						return nil, err
					}
					args = append(args, future.Value())
				}
			} else {
				args = append(args, received.Value())
			}

			// TODO - Do something by looking at the signature of the operator
			// to allow injection for nodes with no inputs or type matching.

			// Call the actual function with the args
			if operator, is := this.(xg.Operator); is {
				return operator.OperatorFunc()(args)
			}
			result := fmt.Sprintf("call_%v(%v)", this.NodeKey(), args)
			return result, nil
		})

		result := work{ctx: w.ctx, id: w.id, from: this, Awaitable: future, callback: w.callback}

		if len(outbound) == 0 {
			// write to the graph's output
			fg.output[this] <- result
			fg.Log("Graph output", "id", w.id, "output", fg.output[this])
		} else {
			// write to downstream nodes
			for _, ch := range outbound {
				ch <- result
			}
			fg.Log("Send to downstream", "id", w.id, "result", result, "outbound", len(outbound))
		}

		// remove from pending
		delete(pending, w.id)
	}
}
*/
