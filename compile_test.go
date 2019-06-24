package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFuncReflectPrototype(t *testing.T) {

	type testType struct {
		name string
	}

	var f interface{}
	var ft reflect.Type

	f = func(int64, int, string, []byte, testType, ...testType) (bool, error) {
		return false, nil
	}
	ft = reflect.TypeOf(f)
	require.Equal(t, reflect.Func, ft.Kind())
	require.True(t, ft.IsVariadic())
	require.Equal(t, 6, ft.NumIn())
	require.Equal(t, 2, ft.NumOut())
	require.Equal(t, reflect.TypeOf(true), ft.Out(0))

	test := []reflect.Type{
		reflect.TypeOf(testType{"t0"}),
		reflect.TypeOf(100),
		reflect.TypeOf("string2"),
		reflect.TypeOf(false),
		reflect.TypeOf("string1"),
		reflect.TypeOf([]byte("hello")),
		reflect.TypeOf([]testType{{"t1"}}),
	}
	t.Log(test)
}

func testOrderByContextIndex(a, b Edge) bool {
	ca := a.Context()
	cb := b.Context()
	if len(ca) > 0 && len(cb) > 0 {
		idx, ok := ca[0].(int)
		if ok {
			idx2, ok2 := cb[0].(int)
			if ok2 {
				return idx < idx2
			}
		}
	}
	return strings.Compare(fmt.Sprintf("%v", a.From().NodeKey()), fmt.Sprintf("%v", b.From().NodeKey())) < 0
}

type flowData map[Node]Awaitable

func from(edges []Edge) []Node {
	result := make([]Node, len(edges))
	for i := range edges {
		result[i] = edges[i].From()
	}
	return result
}

func (m flowData) matches(gen func() []Node) bool {
	matches := 0
	test := gen()
	for _, n := range test {
		_, has := m[n]
		if has {
			matches++
		}
	}
	return len(m) == len(test)
}

func TestCompileExec(t *testing.T) {

	print := func(nodeKey interface{}) OperatorFunc {
		return func(args []interface{}) (interface{}, error) {
			return fmt.Sprintf("%v(%v)", nodeKey, args), nil
		}
	}

	// Ratio(Sum(x1, x2, x3), Sum(x3, y1, y2))
	x1 := &nodeT{id: "x1", operator: print("x1")}
	x2 := &nodeT{id: "x2", operator: print("x2")}
	x3 := &nodeT{id: "x3", operator: print("x3")}
	y1 := &nodeT{id: "y1", operator: print("y1")}
	y2 := &nodeT{id: "y2", operator: print("y2")}
	sumX := &nodeT{id: "sumX", operator: print("sumX")}
	sumY := &nodeT{id: "sumY", operator: print("sumY")}
	ratio := &nodeT{id: "ratio", operator: print("ratio")}

	input := EdgeKind(1)

	g := Builder(Options{})
	g.Add(x1, x2, x3, y1, y2, sumX, sumY, ratio)

	g.Associate(x1, input, sumX) // ordering comes from the nodeKey, lexicographically
	g.Associate(x2, input, sumX)
	g.Associate(x3, input, sumX)
	g.Associate(y1, input, sumY, 2)
	g.Associate(y2, input, sumY, 1)
	g.Associate(x3, input, sumY, 0)
	g.Associate(sumX, input, ratio, 0) // context is the positional arg index
	g.Associate(sumY, input, ratio, 1)

	flowGraph, err := NewFlowGraph(g, input)
	require.NoError(t, err)

	flowGraph.Logger = stdout(0)
	flowGraph.EdgeLessFunc = testOrderByContextIndex

	require.NoError(t, flowGraph.Compile())

	ctx := context.Background()

	require.Equal(t, x1, g.Node(NodeKey("x1")))

	output, err := flowGraph.Run(ctx, map[Node]interface{}{
		g.Node(NodeKey("x1")): "x1v",
		x2:                    "x2v",
		x3:                    "x3v",
		y1:                    "y1v",
		y2:                    "y2v",
	})
	require.NoError(t, err)

	var dag Awaitable = (<-output)[ratio]
	require.NotNil(t, dag)

	require.Equal(t, "ratio([sumX([x1([x1v]) x2([x2v]) x3([x3v])]) sumY([x3([x3v]) y2([y2v]) y1([y1v])])])", dag.Value())
}

type Logger interface {
	Log(...interface{})
}

type stdout int

func (s stdout) Log(args ...interface{}) {
	fmt.Println(args...)
}

type FlowID int64

type work struct {
	Awaitable

	ctx      context.Context
	id       FlowID
	from     Node
	callback chan map[Node]Awaitable
}
type edgeSlice []Edge

func (s edgeSlice) From() (from []Node) {
	from = make([]Node, len(s))
	for i := range s {
		from[i] = s[i].From()
	}
	return
}

type FlowGraph struct {
	Logger
	Graph
	Kind         EdgeKind
	EdgeLessFunc func(a, b Edge) bool // returns True if a < b

	flow       []Node // topological order
	links      []chan work
	input      map[Node]chan<- work
	output     map[Node]chan work
	aggregator chan work
}

func NewFlowGraph(g Graph, kind EdgeKind) (*FlowGraph, error) {
	fg := &FlowGraph{
		Graph:      g,
		Kind:       kind,
		links:      []chan work{},
		input:      map[Node]chan<- work{},
		output:     map[Node]chan work{},
		aggregator: make(chan work),
	}
	flow, err := DirectedSort(g, kind)
	if err != nil {
		return nil, err
	}

	fg.flow = flow
	return fg, nil
}

func (fg *FlowGraph) Run(ctx context.Context, args map[Node]interface{}) (<-chan map[Node]Awaitable, error) {
	callback := make(chan map[Node]Awaitable)
	id := FlowID(time.Now().UnixNano())

	fg.Log(id, "Run with input", args)

	for k, v := range args {
		if ch, has := fg.input[k]; has {
			source := k
			arg := v
			ch <- work{
				ctx:      ctx,
				id:       id,
				from:     source,
				callback: callback,
				Awaitable: Async(ctx, func() (interface{}, error) {
					fg.Log(id, source, "Exec with value=", arg)
					return arg, nil
				}),
			}
		} else {
			return nil, fmt.Errorf("not an input node %v", k)
		}
	}
	return callback, nil
}

func (fg *FlowGraph) Compile() error {

	edgeChannels := map[Edge]chan work{}
	flow := fg.flow

	for i := range flow {

		this := flow[i]

		to := EdgeSlice(fg.To(fg.Kind, this).Edges())
		from := EdgeSlice(fg.From(this, fg.Kind).Edges())

		// Build the output first.  For each output edge
		// we create a work channel for downstream node to receive
		outbound := map[Edge]chan<- work{}
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
		SortEdges(to, fg.EdgeLessFunc)

		// Create links based on input and output edges:

		// For input, we need one more aggregation channel
		// that collects all the input for the given flow id.
		aggregator := make(chan work)
		go func() {

			pending := map[FlowID]flowData{}

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

				if len(to) > 0 && !inputMap.matches(edgeSlice(to).From) {
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

				future := Async(ctx, func() (interface{}, error) {

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
					if operator, is := this.(Operator); is {
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
		pending := map[FlowID]flowData{}
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

				if !output.matches(func() (result []Node) {
					result = []Node{}
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
