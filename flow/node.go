package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"time"

	xg "github.com/orkestr8/xgraph"
)

type graph struct {
	links   links
	input   xg.NodeSlice
	output  xg.NodeSlice
	ordered []*node
}

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

func analyze(g xg.Graph, kind xg.EdgeKind, ordered xg.NodeSlice) (*graph, error) {
	nodes := []*node{}
	links := map[xg.Edge]chan work{}
	graphInput := xg.NodeSlice{}
	graphOutput := xg.NodeSlice{}

	// First pass - build connections
	for _, this := range ordered {
		from := g.From(this, kind).Edges().Slice()
		for _, edge := range from {
			links[edge] = make(chan work)
		}
	}

	// Second pass - build nodes and connect input/output
	for i := range ordered {

		this := ordered[i]

		// Inputs TO the node:
		to := g.To(kind, this).Edges().Slice()
		// Outputs FROM the node:
		from := g.From(this, kind).Edges().Slice()

		collector := make(chan work)
		recv, err := receiveChannels(links, to)
		if err != nil {
			return nil, err
		}
		send, err := sendChannels(links, from)
		if err != nil {
			return nil, err
		}
		node := &node{
			Node: this,
			input: &input{
				edges:   to,
				recv:    recv,
				collect: collector,
			},
			output: &output{
				edges: from,
				send:  send,
			},
		}
		if operator, is := this.(xg.Operator); is {
			node.then = then(operator.OperatorFunc())
		}

		nodes = append(nodes, node)

		if len(to) == 0 {
			// No edges come TO this node, so it's an input node for the graph.
			graphInput = append(graphInput, this)
		}
		if len(from) == 0 {
			// No edges come FROM this node, so it's an output node for the graph.
			graphOutput = append(graphOutput, this)
		}
	}

	return &graph{links: links, input: graphInput, output: graphOutput, ordered: nodes}, nil
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
		ctx, _ := context.WithTimeout(w.ctx, 1*time.Second)
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
