package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"fmt"

	xg "github.com/orkestr8/xgraph"
)

func (fg *FlowGraph) Compile() error {

	edgeChannels := map[xg.Edge]chan work{}
	flow := fg.flow

	for i := range flow {

		this := flow[i]

		to := xg.EdgeSlice(fg.To(fg.Kind, this).Edges())
		from := xg.EdgeSlice(fg.From(this, fg.Kind).Edges())

		// Build the output first.  For each output edge
		// we create a work channel for downstream node to receive
		outbound := map[xg.Edge]chan<- work{}
		for i := range from {
			ch := make(chan work)
			fg.links = append(fg.links, ch)
			outbound[from[i]] = ch
			edgeChannels[from[i]] = ch // to be looked up by downstream
		}
		if len(from) == 0 {
			// This node has no edges to other nodes. So it's terminal
			// so we collect its output to send the graph's collector.
			ch := make(chan work)
			fg.output[this] = ch
			go func() {
				for {
					w, ok := <-ch
					if !ok {
						return
					}
					fg.aggregator <- w
				}
			}()
		}

		// Sort the edges by context[0]
		xg.SortEdges(to, fg.EdgeLessFunc)

		// Create links based on input and output edges:

		// For input, we need one more aggregation channel
		// that collects all the input for the given flow id.
		aggregator := make(chan work)
		go func() {

			pending := map[flowID]flowData{}

		node_aggregator:
			for {
				w, ok := <-aggregator
				if !ok {
					return
				}
				fg.Log(w.id, this, "Got work", w)
				// match messages by flow id.
				inputMap, has := pending[w.id]
				if !has {
					inputMap = flowData{}
					pending[w.id] = inputMap
				}
				if _, has := inputMap[w.from]; has {
					fg.Log(w.id, this, "Duplicate awaitable for", w.from, "Ignored. input=", inputMap)
				}
				inputMap[w.from] = w

				if len(to) > 0 && !inputMap.matches(edgeSlice(to).from) {
					// Nothing to do... just wait for message to come
					fg.Log(w.id, this, "Keep waiting for more")
					continue node_aggregator
				}

				fg.Log(w.id, this, "Got all input", inputMap, "given", to)

				// Now inputs are collected.  Build another future and pass it on.
				// TODO - context with timeout
				if w.ctx == nil {
					panic("nil ctx")
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
								// TODO - chain errors
								fg.Log(w.id, this, "Running and got error", err)
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
					fg.Log(w.id, this, "Returning result", result)
					return result, nil
				})

				result := work{ctx: w.ctx, id: w.id, from: this, Awaitable: future, callback: w.callback}

				if len(outbound) == 0 {
					fg.Log(w.id, this, "Sending graph output", result, "output", fg.output[this])
					// write to the graph's output
					fg.output[this] <- result
					fg.Log(w.id, this, "Sent graph output")
				} else {
					// write to downstream nodes
					fg.Log(w.id, this, "Sending result downstream", result)
					for _, ch := range outbound {
						ch <- result
					}
				}

				// remove from pending
				delete(pending, w.id)
			}
		}()

		if len(to) == 0 {
			// No input means this is a Source node whose computation will be input to others
			// So this is an input node for the graph.
			inputChan := make(chan work)
			fg.input[this] = inputChan

			// This input channel will send work directly to the aggregator
			go func() {
				for {
					w, ok := <-inputChan
					if !ok {
						return
					}
					aggregator <- w
				}
			}()
		} else {
			// For each input edge, we should have already created
			// the channel to send work, because the nodes are topologically sorted.
			for i := range to {

				ch, has := edgeChannels[to[i]]
				if !has {
					return fmt.Errorf("No work channel for inbound edge: %v", to[i])
				}
				// Start receiving from input
				go func() {
					for {
						w, ok := <-ch
						if !ok {
							return
						}
						aggregator <- w
					}
				}()
			}
		}

		fg.Log("COMPILE STEP", this, "IN=", to, "OUT=", from)
	}

	// Start the aggregator
	go func() {
		pending := map[flowID]flowData{}
	graph_aggregator:
		for {
			w, ok := <-fg.aggregator
			if !ok {
				return
			}

			fg.Log(w.id, fg.output, "Graph aggreagator got work", w)

			// If there are multiple output nodes then we have to collect.
			output := pending[w.id]

			if len(fg.output) > 0 {

				if output == nil {
					output = flowData{
						w.from: w,
					}
					pending[w.id] = output
				}

				if !output.matches(func() (result []xg.Node) {
					result = []xg.Node{}
					for k := range fg.output {
						result = append(result, k)
					}
					return
				}) {
					continue graph_aggregator
				}
			}

			fg.Log(w.id, "Collected all outputs", output)
			delete(pending, w.id)
			fg.Log(w.id, "Sending graph output", output)
			if w.callback == nil {
				panic("nil callback")
			}
			w.callback <- output
			fg.Log(w.id, "Sent output", output)
			close(w.callback)
		}
	}()
	return nil
}
