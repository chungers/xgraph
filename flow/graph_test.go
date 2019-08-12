package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"testing"
	"time"

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
		Logger: testlog{t},
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

	ctx, result, err := g.execValues(context.Background(),
		map[xg.Node]interface{}{
			x1: "X1",
			x2: "X2",
			x3: "X3",
			y1: "Y1",
			y2: "Y2",
		})
	require.NoError(t, err)
	require.NotNil(t, gatherChanFrom(ctx))
	require.NotNil(t, flowIDFrom(ctx))

	m := map[xg.Node]Awaitable(<-result)
	require.NotNil(t, m[ratio])
	require.Equal(t, 1, len(m))

	require.Equal(t, "ratio([sumX([X1 X2 X3]) sumY([X3 Y2 Y1])])", m[ratio].Value())

	require.NoError(t, g.Close())
}

func TestGraphExecPartialTimeout(t *testing.T) {
	g, gg, _ := testAnalyzeGraph(t)
	g.run()

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	// Partial inputs.  Since input is partial,
	// the futures will block indefinitely until
	// values are met, unless there is timeout or
	// cancellation.
	X := map[xg.Node]interface{}{
		x1: "X1",
		x2: "X2",
		x3: "X3",
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	ctx, result, err := g.execValues(ctx, X)
	require.NoError(t, err)

	select {

	case <-time.After(2 * time.Second):
		t.Fail()

	case m := <-result:
		// result should timeout
		require.Error(t, map[xg.Node]Awaitable(m)[ratio].Error())
	}
	require.NoError(t, g.Close())
}

func TestGraphExecPartialLateComplete(t *testing.T) {
	g, gg, _ := testAnalyzeGraph(t)
	g.run()

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	y1 := gg.Node(xg.NodeKey("y1"))
	y2 := gg.Node(xg.NodeKey("y2"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	// Partial inputs.  Since input is partial,
	// the futures will block indefinitely until
	// values are met.
	X := map[xg.Node]interface{}{
		x1: "X1",
		x2: "X2",
		x3: "X3",
	}

	// No cancelation..  will block indefinitely.
	ctx := context.Background()
	ctx, result, err := g.execValues(ctx, X)
	require.NoError(t, err)

	done := make(chan interface{})
	go func() {
		defer close(done)

		select {

		case <-time.After(2 * time.Second):
			t.Fail()

		case m := <-result:
			// result should timeout
			require.NoError(t, map[xg.Node]Awaitable(m)[ratio].Error())

			done <- map[xg.Node]Awaitable(m)[ratio].Value()
		}
	}()

	// Send in the rest...
	ctx, _, err = g.execValues(ctx, map[xg.Node]interface{}{
		y1: "Y1",
	})
	require.NoError(t, err)

	// Send in the rest...
	ctx, _, err = g.execValues(ctx, map[xg.Node]interface{}{
		y2: "Y2",
	})
	require.NoError(t, err)

	res := <-done

	require.Equal(t, "ratio([sumX([X1 X2 X3]) sumY([X3 Y2 Y1])])", res)

	require.NoError(t, g.Close())
}
