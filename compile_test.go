package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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

	flow, err := DirectedSort(g, input)
	require.NoError(t, err)
	t.Log(flow)

	futures := map[Edge]Awaitable{}
	flowStart := []chan<- interface{}{}
	flowInput := map[Node]chan<- interface{}{}
	flowOperators := []*future{}

	for i := range flow {

		this := flow[i]

		to := EdgeSlice(g.To(input, this).Edges())
		from := EdgeSlice(g.From(this, input).Edges())

		// Sort the edges by context[0]
		SortEdges(to, testOrderByContextIndex)

		input := []Awaitable{}
		for _, in := range to {
			if f, has := futures[in]; has {
				input = append(input, f)
			}
		}

		// No input means this is a Source node whose computation will be input to others
		var startChan, inputChan chan interface{}
		if len(input) == 0 {
			inputChan = make(chan interface{}, 1)
			flowInput[this] = inputChan

			startChan = make(chan interface{})
			flowStart = append(flowStart, startChan)
		}

		t.Log("COMPILE STEP", this, "IN=", to, "OUT=", from, "INPUT=", input)

		f := newFuture(func() (interface{}, error) {

			if startChan != nil {
				<-startChan
			}

			// Given input in array of awaitable...
			args := []interface{}{}

			for i := range input {
				if err := input[i].Error(); err != nil {
					return nil, err
				} else {
					args = append(args, input[i].Value())
				}
			}

			// t.Log("RUNNING - CUR", this, "IN=", to, "OUT=", from, "ARGS=", args)

			// Call the actual function
			// Just print the operator
			out := fmt.Sprintf("%v", this.NodeKey())
			if len(input) > 0 {
				out = fmt.Sprintf("%v( %v )", this.NodeKey(), args)
			} else if inputChan != nil {

				defer close(inputChan)

				v := <-inputChan
				out = fmt.Sprintf("%v", v)
			}

			return out, nil
		})

		for _, out := range from {
			futures[out] = f
		}

		flowOperators = append(flowOperators, f)
	}

	// Now we've built the graph.  Execute it.
	done := make(chan interface{})
	go func() {
		t.Log("result=", flowOperators[len(flowOperators)-1].Value())
		close(done)
	}()

	require.Equal(t, 5, len(flowStart))

	ctx := context.Background()

	// Start processing
	for i := range flowOperators {
		flowOperators[i].doAsync(ctx)
	}

	// Give signal to start
	for i := range flowStart {
		close(flowStart[i])
	}

	// Set the input
	require.Equal(t, 5, len(flowInput))
	require.NotNil(t, flowInput[x1])
	require.NotNil(t, flowInput[x2])
	require.NotNil(t, flowInput[x3])
	require.NotNil(t, flowInput[y1])
	require.NotNil(t, flowInput[y2])

	// Idea: take a map and set the values accordingly
	flowInput[x1] <- "x1v"
	flowInput[x2] <- "x2v"
	flowInput[x3] <- "x3v"
	flowInput[y1] <- "y1v"
	flowInput[y2] <- "y2v"

	<-done

	var dag Awaitable = flowOperators[len(flowOperators)-1]

	require.Equal(t, "ratio( [sumX( [x1v x2v x3v] ) sumY( [x3v y2v y1v] )] )", dag.Value())
}
