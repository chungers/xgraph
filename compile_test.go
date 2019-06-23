package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

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

	require.NoError(t, flowGraph.Run(ctx))

	// Now we've built the graph.  Execute it.
	done := make(chan interface{})
	go func() {
		for k, v := range flowGraph.Output {
			t.Log("result[", k, "] = ", v.Value())
		}
		close(done)
	}()

	// Set the input
	require.Equal(t, 5, len(flowGraph.Input))
	for _, n := range []Node{x1, x2, x3, y1, y2} {
		require.NotNil(t, flowGraph.Input[n])
	}

	t.Log("Setting input")

	// Idea: take a map and set the values accordingly
	flowGraph.SetInput(map[Node]interface{}{
		x1: "x1v",
		x2: "x2v",
		x3: "x3v",
		y1: "y1v",
		y2: "y2v",
	})

	<-done

	t.Log("Done execute graph")

	var dag Awaitable = flowGraph.Output[ratio]
	require.NotNil(t, dag)

	require.Equal(t, "ratio( [sumX( [x1v x2v x3v] ) sumY( [x3v y2v y1v] )] )", dag.Value())
}

type Logger interface {
	Log(...interface{})
}

type FlowGraph struct {
	Logger
	Graph
	Kind         EdgeKind
	Input        map[Node]chan<- interface{}
	Output       map[Node]Awaitable
	EdgeLessFunc func(a, b Edge) bool // returns True if a < b

	flow      []Node // topological order
	runnables []*future
}

func NewFlowGraph(g Graph, kind EdgeKind) (*FlowGraph, error) {
	fg := &FlowGraph{
		Graph:  g,
		Kind:   kind,
		Input:  map[Node]chan<- interface{}{},
		Output: map[Node]Awaitable{},
	}
	flow, err := DirectedSort(g, kind)
	if err != nil {
		return nil, err
	}

	fg.flow = flow
	return fg, nil
}

func (fg *FlowGraph) SetInput(m map[Node]interface{}) {
	for k, v := range m {
		if ch, has := fg.Input[k]; has {
			ch <- v
		}
	}
}

func (fg *FlowGraph) Run(ctx context.Context) error {
	if len(fg.runnables) == 0 {
		return fmt.Errorf("no futures")
	}
	for i := range fg.runnables {
		fg.runnables[i].doAsync(ctx)
	}
	return nil
}

func (fg *FlowGraph) Compile() error {
	futures := map[Edge]Awaitable{}
	runnables := []*future{}

	flow := fg.flow

	for i := range flow {

		this := flow[i]

		to := EdgeSlice(fg.To(fg.Kind, this).Edges())
		from := EdgeSlice(fg.From(this, fg.Kind).Edges())

		// No input means this is a Source node whose computation will be input to others
		// So this is an input node for the graph.
		var inputChan chan interface{}
		if len(to) == 0 {
			inputChan = make(chan interface{}, 1)
			fg.Input[this] = inputChan
		}

		// Sort the edges by context[0]
		SortEdges(to, testOrderByContextIndex)

		fg.Log("COMPILE STEP", this, "IN=", to, "OUT=", from)

		nodeOperator := func() (interface{}, error) {

			input := []Awaitable{}
			for _, in := range to {
				if f, has := futures[in]; has {
					input = append(input, f)
				}
			}

			fg.Log("EXEC STEP", this, "IN=", to, "OUT=", from, "INPUT=", input)

			// Given input in array of awaitable...
			args := []interface{}{}

			for i := range input {
				if err := input[i].Error(); err != nil {
					return nil, err
				} else {
					args = append(args, input[i].Value())
				}
			}

			// Call the actual function
			// Just print the operator
			out := fmt.Sprintf("%v", this.NodeKey())
			if len(args) > 0 {
				out = fmt.Sprintf("%v( %v )", this.NodeKey(), args)
			} else if inputChan != nil {

				defer close(inputChan)

				v := <-inputChan
				out = fmt.Sprintf("%v", v)
			}

			return out, nil
		}

		f := newFuture(nodeOperator)

		// Index this node's output by outbound Edge,
		// so nodes down stream can use as input
		for _, out := range from {
			futures[out] = f
		}

		runnables = append(runnables, f)

		// No output so this node is the terminal node in the graph.
		if len(from) == 0 {
			fg.Output[this] = f
		}
	}

	fg.runnables = runnables
	return nil
}
