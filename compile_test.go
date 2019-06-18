package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOrderByContextIndex(a, b Edge) bool {
	if a.To().NodeKey() != b.To().NodeKey() {
		return false
	}
	ca := a.Context()
	cb := b.Context()
	if len(ca) == 0 && len(cb) == 0 {
		return false
	}
	idx, ok := ca[0].(int)
	if ok {
		idx2, ok2 := cb[0].(int)
		if ok2 {
			return idx < idx2
		}
	}
	return false
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

	g.Associate(x1, input, sumX, 0)
	g.Associate(x2, input, sumX, 1)
	g.Associate(x3, input, sumX, 2)
	g.Associate(y1, input, sumY, 2)
	g.Associate(y2, input, sumY, 1)
	g.Associate(x3, input, sumY, 0)
	g.Associate(sumX, input, ratio, 0) // context is the positional arg index
	g.Associate(sumY, input, ratio, 1)

	flow, err := DirectedSort(g, input)
	require.NoError(t, err)
	t.Log(flow)

	ctx := context.Background()

	futures := map[Edge]Awaitable{}

	var dag Awaitable

	for i := range flow {

		to := EdgeSlice(g.To(input, flow[i]).Edges())
		from := EdgeSlice(g.From(flow[i], input).Edges())

		// Sort the edges by context[0]
		SortEdges(to, testOrderByContextIndex)

		input := []Awaitable{}
		for _, in := range to {
			if f, has := futures[in]; has {
				input = append(input, f)
			}
		}

		t.Log("COMPILE STEP", flow[i], "IN=", to, "OUT=", from, "INPUT=", input)

		this := flow[i]

		f := Async(ctx,
			func() (interface{}, error) {
				// Given input in array of awaitable...
				args := []interface{}{}

				for i := range input {
					if err := input[i].Error(); err != nil {
						return nil, err
					} else {
						args = append(args, input[i].Value())
					}
				}

				// t.Log("RUNNING - CUR", flow[i], "IN=", to, "OUT=", from, "ARGS=", args)

				// Call the actual function
				// Just print the operator
				out := fmt.Sprintf("%v", this.NodeKey())
				if len(args) > 0 {
					out = fmt.Sprintf("%v( %v )", this.NodeKey(), args)
				}

				return out, nil
			})

		for _, out := range from {
			futures[out] = f
		}

		dag = f
	}

	t.Log("result=", dag.Value())
	require.Equal(t, "ratio( [sumX( [x1 x2 x3] ) sumY( [x3 y2 y1] )] )", dag.Value())
}
