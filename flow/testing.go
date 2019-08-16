package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"fmt"

	xg "github.com/orkestr8/xgraph"
)

func testBuildGraph(input xg.EdgeKind, inline ...bool) xg.Graph {

	print := func(nodeKey interface{}) xg.OperatorFunc {
		return func(args []interface{}) (interface{}, error) {
			return fmt.Sprintf("%v(%v)", nodeKey, args), nil
		}
	}

	attrs := map[string]interface{}{}
	if len(inline) > 0 && inline[0] {
		attrs["inline"] = true
	}

	// Ratio(Sum(x1, x2, x3), Sum(x3, y1, y2))
	x1 := &nodeT{id: "x1", operator: print("x1"), attributes: attrs}
	x2 := &nodeT{id: "x2", operator: print("x2"), attributes: attrs}
	x3 := &nodeT{id: "x3", operator: print("x3"), attributes: attrs}
	y1 := &nodeT{id: "y1", operator: print("y1"), attributes: attrs}
	y2 := &nodeT{id: "y2", operator: print("y2"), attributes: attrs}
	sumX := &nodeT{id: "sumX", operator: print("sumX"), attributes: attrs}
	sumY := &nodeT{id: "sumY", operator: print("sumY"), attributes: attrs}
	ratio := &nodeT{id: "ratio", operator: print("ratio"), attributes: attrs}

	g := xg.Builder(xg.Options{})
	g.Add(x1, x2, x3, y1, y2, sumX, sumY, ratio)

	g.Associate(x1, input, sumX) // ordering comes from the nodeKey, lexicographically
	g.Associate(x2, input, sumX)
	g.Associate(x3, input, sumX)
	g.Associate(y1, input, sumY, xg.Attribute{Key: "arg", Value: 2})
	g.Associate(y2, input, sumY, xg.Attribute{Key: "arg", Value: 1})
	g.Associate(x3, input, sumY, xg.Attribute{Key: "arg", Value: 0})
	g.Associate(sumX, input, ratio, xg.Attribute{Key: "arg", Value: 0}) // positional arg index
	g.Associate(sumY, input, ratio, xg.Attribute{Key: "arg", Value: 1})
	return g
}
