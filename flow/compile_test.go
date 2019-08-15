package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

func TestV1CompileExec(t *testing.T) {

	input := xg.EdgeKind(1)
	g := testBuildGraph(input)

	flowGraph, err := NewFlowGraph(g, input)
	require.NoError(t, err)

	flowGraph.Logger = nologging{}
	flowGraph.EdgeLessFunc = testOrderByContextIndex

	require.NoError(t, flowGraph.Compile())
}

func BenchmarkV1Compile(b *testing.B) {

	input := xg.EdgeKind(1)
	g := testBuildGraph(input)

	flowGraph, err := NewFlowGraph(g, input)
	if err != nil {
		panic(err)
	}

	flowGraph.Logger = nologging{}
	flowGraph.EdgeLessFunc = testOrderByContextIndex

	for i := 0; i < b.N; i++ {
		err = flowGraph.Compile()
		if err != nil {
			panic(err)
		}
	}
}
