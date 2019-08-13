package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"fmt"
	"testing"

	xg "github.com/orkestr8/xgraph"
)

const (
	// On slower machines testing.T.Log() fails when `go test -race`.
	useTestingLog = true
)

type testlog struct {
	*testing.T
}

func (s testlog) Log(context string, args ...interface{}) {
	if useTestingLog {
		s.T.Log([]interface{}{"INFO", context, args}...)
		return
	}
	fmt.Println([]interface{}{"INFO", context, args}...)
}

func (s testlog) Warn(context string, args ...interface{}) {
	if useTestingLog {
		s.T.Log([]interface{}{"WARN", context, args}...)
	}
	fmt.Println([]interface{}{"WARN", context, args}...)
}

type benchlog struct {
	*testing.B
	log bool
}

func (s benchlog) Log(context string, args ...interface{}) {
	if s.log {
		s.B.Log([]interface{}{"INFO", context, args}...)
	}
}

func (s benchlog) Warn(context string, args ...interface{}) {
	if s.log {
		s.B.Log([]interface{}{"WARN", context, args}...)
	}
}

func testBuildGraph(input xg.EdgeKind) xg.Graph {

	print := func(nodeKey interface{}) xg.OperatorFunc {
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

	g := xg.Builder(xg.Options{})
	g.Add(x1, x2, x3, y1, y2, sumX, sumY, ratio)

	g.Associate(x1, input, sumX) // ordering comes from the nodeKey, lexicographically
	g.Associate(x2, input, sumX)
	g.Associate(x3, input, sumX)
	g.Associate(y1, input, sumY, xg.Attribute{Key: "order", Value: 2})
	g.Associate(y2, input, sumY, xg.Attribute{Key: "order", Value: 1})
	g.Associate(x3, input, sumY, xg.Attribute{Key: "order", Value: 0})
	g.Associate(sumX, input, ratio, xg.Attribute{Key: "order", Value: 0}) // positional arg index
	g.Associate(sumY, input, ratio, xg.Attribute{Key: "order", Value: 1})
	return g
}
