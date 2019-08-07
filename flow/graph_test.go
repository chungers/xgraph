package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

func testAnalyzeGraph(t *testing.T) (*graph, xg.Graph, xg.EdgeKind) {
	deps := xg.EdgeKind(1)
	gg := testBuildGraph(deps)

	ref := GraphRef("test1")
	ordered, err := xg.DirectedSort(gg, deps)
	require.NoError(t, err)

	options := Options{
		Logger: logger(1),
	}
	g, err := analyze(ref, gg, deps, ordered, options)
	require.NoError(t, err)
	return g, gg, deps
}

func TestGraphRunAndClose(t *testing.T) {
	g, _, _ := testAnalyzeGraph(t)
	g.run()
	require.NoError(t, g.Close())
}

func TestGraphExec(t *testing.T) {
	g, gg, _ := testAnalyzeGraph(t)
	g.run()

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	y1 := gg.Node(xg.NodeKey("y1"))
	y2 := gg.Node(xg.NodeKey("y2"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	result, err := g.exec(context.Background(),
		map[xg.Node]interface{}{
			x1: "X1",
			x2: "X2",
			x3: "X3",
			y1: "Y1",
			y2: "Y2",
		})
	require.NoError(t, err)

	m := map[xg.Node]Awaitable(<-result)
	require.NotNil(t, m[ratio])

	t.Log(m[ratio].Value())

	require.NoError(t, g.Close())
}
