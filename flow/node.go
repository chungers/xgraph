package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"time"

	xg "github.com/orkestr8/xgraph"
)

func (node *node) run() {
	node.input.run()
	go node.loop()
}

func (node *node) close() {
	node.input.close()
}

func (node *node) loop() {

	pending := map[flowID]gather{}

	for {
		w, ok := <-node.input.collect
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

		if len(node.input.edges) > 0 && !gathered.hasKeys(xg.EdgeSlice(node.input.edges).FromNodes) {
			// Nothing to do... just wait for message to come
			continue
		}

		w.Log("All input received", "id", w.id, "input", gathered, "given", node.input.edges)

		// Build Future here
		ctx, _ := context.WithTimeout(w.ctx, time.Duration(node.attributes.Timeout))
		future := xg.Async(ctx, func() (interface{}, error) {
			args, err := gathered.args(ctx, node.input.edges)
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
