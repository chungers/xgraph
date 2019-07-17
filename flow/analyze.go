package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

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
			Node:       this,
			attributes: &attributes{},
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
		if attributer, is := this.(xg.Attributer); is {
			attr := &attributes{}
			if err := attr.unmarshal(attributer.Attributes()); err != nil {
				return nil, err
			}
			node.attributes = attr
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
