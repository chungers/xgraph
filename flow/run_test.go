package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

// TODO - This one has a race.  Turn on go test -v -race
func TestV1Run(t *testing.T) {

	input := xg.EdgeKind(1)
	g := testBuildGraph(input)

	flowGraph, err := NewFlowGraph(g, input)
	require.NoError(t, err)

	flowGraph.Logger = nologging{}
	flowGraph.EdgeLessFunc = testOrderByContextIndex

	require.NoError(t, flowGraph.Compile())

	ctx := context.Background()

	output, err := flowGraph.Run(ctx, map[xg.Node]interface{}{
		g.Node(xg.NodeKey("x1")): "x1v",
		g.Node("x2"):             "x2v",
		g.Node("x3"):             "x3v",
		g.Node("y1"):             "y1v",
		g.Node("y2"):             "y2v",
	})
	require.NoError(t, err)

	var dag Awaitable = (<-output)[g.Node("ratio")]
	require.NotNil(t, dag)

	require.Equal(t, "ratio([sumX([x1([x1v]) x2([x2v]) x3([x3v])]) sumY([x3([x3v]) y2([y2v]) y1([y1v])])])", dag.Value())
}

func BenchmarkV1Run(b *testing.B) {
	input := xg.EdgeKind(1)
	g := testBuildGraph(input)

	flowGraph, err := NewFlowGraph(g, input)
	if err != nil {
		panic(err)
	}

	flowGraph.Logger = nologging{}
	flowGraph.EdgeLessFunc = testOrderByContextIndex

	err = flowGraph.Compile()
	if err != nil {
		panic(err)
	}

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		output, err := flowGraph.Run(ctx, map[xg.Node]interface{}{
			g.Node(xg.NodeKey("x1")): "x1v",
			g.Node("x2"):             "x2v",
			g.Node("x3"):             "x3v",
			g.Node("y1"):             "y1v",
			g.Node("y2"):             "y2v",
		})
		if err != nil {
			panic(err)
		}

		var dag Awaitable = (<-output)[g.Node("ratio")]
		dag.Value()
	}
}
