package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"time"

	xg "github.com/orkestr8/xgraph"
)

func sendChannels(links links, edges xg.EdgeSlice) ([]chan<- work, error) {
	out := []chan<- work{}
	for _, edge := range edges {
		c, has := links[edge]
		if !has {
			return nil, fmt.Errorf("No channel allocated for edge %v", edge)
		}
		out = append(out, c)
	}
	return out, nil
}

func receiveChannels(links links, edges xg.EdgeSlice) ([]<-chan work, error) {
	out := []<-chan work{}
	for _, edge := range edges {
		c, has := links[edge]
		if !has {
			return nil, fmt.Errorf("No channel allocated for edge %v", edge)
		}
		out = append(out, c)
	}
	return out, nil
}

func (input *input) run() {
	for _, c := range input.recv {
		go func() {
			for {
				w, ok := <-c
				if !ok {
					return
				}
				input.collect <- w
			}
		}()
	}
}

func (input *input) close() {
	close(input.collect)
}

func (output *output) dispatch(w work) {
	for _, c := range output.send {
		c <- w
	}
}

func (node *node) close() {
	node.input.close()
}

func (node *node) loop() {

	pending := map[flowID]flowData{}

	for {
		w, ok := <-node.input.collect
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

		if len(node.input.edges) > 0 && !fd.hasKeys(xg.EdgeSlice(node.input.edges).FromNodes) {
			// Nothing to do... just wait for message to come
			continue
		}

		w.Log("All input received", "id", w.id, "input", fd, "given", node.input.edges)

		// Build Future here
		ctx, _ := context.WithTimeout(w.ctx, time.Duration(node.attributes.Timeout))
		future := xg.Async(ctx, func() (interface{}, error) {
			args, err := fd.args(ctx, node.input.edges)
			if err != nil {
				return nil, err
			}
			return node.then(args) // TODO - also pass in ctx?
		})

		// Scatter / dispatch work
		node.output.dispatch(work{ctx: w.ctx, id: w.id, from: node.Node, Awaitable: future, callback: w.callback})

		// remove from pending list
		delete(pending, w.id)
	}
}
