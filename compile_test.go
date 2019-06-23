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

func TestCompileExec(t *testing.T) {

	// Ratio(Sum(x1, x2, x3), Sum(x3, y1, y2))
	x1 := &nodeT{id: "x1"}
	x2 := &nodeT{id: "x2"}
	x3 := &nodeT{id: "x3"}
	y1 := &nodeT{id: "y1"}
	y2 := &nodeT{id: "y2"}
	sumX := &nodeT{id: "sumX"}
	sumY := &nodeT{id: "sumY"}
	ratio := &nodeT{id: "ratio"}

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

	flowGraph.Logger = t
	flowGraph.EdgeLessFunc = testOrderByContextIndex

	require.NoError(t, flowGraph.Compile())

	ctx := context.Background()

	output, err := flowGraph.Run(ctx, map[Node]interface{}{
		x1: "x1v",
		x2: "x2v",
		x3: "x3v",
		y1: "y1v",
		y2: "y2v",
	})
	require.NoError(t, err)

	var dag Awaitable = (<-output)[ratio]
	require.NotNil(t, dag)

	require.Equal(t, "ratio( [sumX( [x1v x2v x3v] ) sumY( [x3v y2v y1v] )] )", dag.Value())
}

type Logger interface {
	Log(...interface{})
}

type FlowID int64

type work struct {
	Awaitable

	ctx      context.Context
	id       FlowID
	from     Node
	callback chan map[Node]Awaitable
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
	for k, v := range args {
		if ch, has := fg.input[k]; has {
			ch <- work{
				ctx:       ctx,
				id:        id,
				callback:  callback,
				Awaitable: Async(ctx, func() (interface{}, error) { return v, nil }),
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
			ch := make(chan work, 1)
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

			pending := map[FlowID]map[Node]Awaitable{}
		node_aggregator:
			for {
				w, ok := <-aggregator
				if !ok {
					return
				}
				// match messages by flow id.
				inputMap, has := pending[w.id]
				if !has {
					inputMap = map[Node]Awaitable{w.from: w}
					pending[w.id] = inputMap
				}
				if _, has := inputMap[w.from]; has {
					fg.Log("Duplicate awaitable for %v. Ignored.", w.from)
					continue node_aggregator
				}
				inputMap[w.from] = w
				// Now check for all input are present. If so, build the output
				matches := 0
				for i := range to {
					if _, has := inputMap[to[i].From()]; has {
						matches++
					}
				}
				if matches != len(to) {
					// Nothing to do... just wait for message to come
					continue node_aggregator
				}
				// Now inputs are collected.  Build another future and pass it on.
				// TODO - context with timeout
				ctx := w.ctx

				future := Async(ctx, func() (interface{}, error) {

					// Wait for all inputs to complete computation and build args
					// for this node before proceeding with this node's computation.
					args := []interface{}{}
					for i := range to {
						future := inputMap[to[i].From()]

						// Calling the Value or Error will block until completion
						if err := future.Error(); err != nil {
							// TODO - chain errors
							return nil, err
						}
						args = append(args, future.Value())
					}

					// Call the actual function with the args
					if len(args) == 0 {
						return this.NodeKey(), nil
					}
					return fmt.Sprintf("%v(%v)", this.NodeKey(), args), nil
				})

				result := work{id: w.id, from: this, Awaitable: future, callback: w.callback}

				if len(outbound) == 0 {
					// write to the graph's output
					fg.output[this] <- result
				} else {
					// write to downstream nodes
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
			inputChan := make(chan work, 1)
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
		pending := map[FlowID]map[Node]Awaitable{}
	graph_aggregator:
		for {
			w, ok := <-fg.aggregator
			if !ok {
				return
			}

			output, has := pending[w.id]
			if !has {
				pending[w.id] = map[Node]Awaitable{
					w.from: w,
				}
				continue graph_aggregator
			}

			// check completion
			match := 0
			for k := range fg.output {
				if _, has := output[k]; has {
					match++
				}
			}
			if match == len(fg.output) {
				delete(pending, w.id)
				w.callback <- output
			}
		}
	}()
	return nil
}
