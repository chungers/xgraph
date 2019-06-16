package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileExec(t *testing.T) {

	// Ratio(Sum(x1, x2, x3), Sum(y1, y2, x3))
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

	g.Associate(x1, input, sumX)
	g.Associate(x2, input, sumX)
	g.Associate(x3, input, sumX)
	g.Associate(y1, input, sumY)
	g.Associate(y2, input, sumY)
	g.Associate(x3, input, sumY)
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

		t.Log("CUR", flow[i], "IN=", to, "OUT=", from)

		// Sort the edges by context[0]

		input := []Awaitable{}
		for _, in := range to {
			if f, has := futures[in]; has {
				input = append(input, f)
			}
		}

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

				// call the actual function

				return nil, nil
			})

		for _, out := range from {
			futures[out] = f
		}

		dag = f
	}
}
