package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"fmt"
	"strings"

	xg "github.com/orkestr8/xgraph"
	"golang.org/x/sync/semaphore"
)

func OrderEdgesByEdgeAttributeOrderOrNodeKey(a, b xg.Edge) bool {
	ca := a.Attributes()
	cb := b.Attributes()
	if len(ca) > 0 && len(cb) > 0 {
		idx, ok := ca["arg"].(int)
		if ok {
			idx2, ok2 := cb["arg"].(int)
			if ok2 {
				return idx < idx2
			}
		}
	}
	return strings.Compare(fmt.Sprintf("%v", a.From().NodeKey()), fmt.Sprintf("%v", b.From().NodeKey())) < 0
}

func analyze(ref GraphRef, g xg.Graph, kind xg.EdgeKind, ordered xg.NodeSlice,
	options Options) (*graph, error) {

	if options.Logger == nil {
		options.Logger = nologging{}
	}

	nodes := []*node{}
	links := map[xg.Edge]chan work{}
	graphInput := map[xg.Node]chan<- work{}
	graphOutput := map[xg.Node]<-chan work{}

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

		attr := attributes{}
		if attributer, is := this.(xg.Attributer); is {
			if err := attr.unmarshal(attributer.Attributes()); err != nil {
				return nil, err
			}
		}

		// Inputs TO the node:
		to := g.To(kind, this).Edges().Slice()
		// Outputs FROM the node:
		from := g.From(this, kind).Edges().Slice()

		// Sort the TO edges (input edges)
		// TODO(dchung) - implement sorter based on reflection of the operator function
		xg.SortEdges(to, edgeSorter(attr.EdgeSorter))

		collect := make(chan work)
		inbound, err := receiveChannels(links, to)
		if err != nil {
			return nil, err
		}
		outbound, err := sendChannels(links, from)
		if err != nil {
			return nil, err
		}

		inputFromNodes := to.FromNodes()
		outputToNodes := from.ToNodes()

		// TODO(dchung) - Check to see if this node is a reference to another graph.
		// if so, we can wire the input/output directly instead of creating a node instance.

		options.Log("Building node", "node", this, "inputChans", inbound, "outputChans", outbound,
			"inputEdges", to, "outputEdges", from)
		node := &node{
			Node:       this,
			Logger:     options.Logger,
			attributes: attr,
			collect:    collect,
			inbound:    inbound,
			outbound:   outbound,
			inputFrom:  func() xg.NodeSlice { return inputFromNodes },
			outputTo:   func() xg.NodeSlice { return outputToNodes },
			stop:       make(chan interface{}),
		}

		node.defaults() // default fields if not set

		if operator, is := this.(xg.Operator); is {
			node.then = then(operator.OperatorFunc())
		}

		if node.attributes.MaxWorkers > 0 {
			node.sem = semaphore.NewWeighted(int64(node.attributes.MaxWorkers))
		}

		if len(to) == 0 {
			// No edges come TO this node, so it's an input node for the graph.
			graphInput[this] = node.collect
		}
		if len(from) == 0 {
			// No edges come FROM this node, so it's an output node for the graph.
			if len(node.outbound) > 0 {
				panic(fmt.Errorf("No outputTo nodes but allocated output channels: %v", this))
			}
			// we should have a collection point
			ch := make(chan work)
			node.outbound = []chan<- work{ch}
			graphOutput[this] = ch
		}

		nodes = append(nodes, node)
	}

	// Last node is the collection of all terminal output nodes.
	// so a graph of N nodes will have N+1 nodes allocated.  It will always be the last one.
	// Use a node implementation to collect all the futures from output nodes
	gOutput, gOutputChs := pairs(graphOutput)
	agg := (&node{
		terminal:  true,
		Node:      ref,
		Logger:    options.Logger,
		collect:   make(chan work),
		inbound:   gOutputChs,
		inputFrom: func() xg.NodeSlice { return gOutput },
		stop:      make(chan interface{}),
	}).defaults()

	nodes = append(nodes, agg)

	return &graph{Node: ref, input: graphInput, output: graphOutput, ordered: nodes}, nil
}

func pairs(m map[xg.Node]<-chan work) (keys xg.NodeSlice, chs []<-chan work) {
	keys = xg.NodeSlice{}
	chs = []<-chan work{}
	for k, v := range m {
		keys = append(keys, k)
		chs = append(chs, v)
	}
	return
}
