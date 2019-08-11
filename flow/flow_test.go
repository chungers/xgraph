package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

func TestFlowNew(t *testing.T) {
	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind)
	options := Options{
		Logger: testlog{t},
	}
	executor, err := New(ref, gg, kind, options)
	require.NoError(t, err)

	require.NoError(t, executor.Close())
}

func TestFlowExecFull(t *testing.T) {
	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind)
	options := Options{
		Logger: testlog{t},
	}
	executor, err := New(ref, gg, kind, options)
	require.NoError(t, err)

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	y1 := gg.Node(xg.NodeKey("y1"))
	y2 := gg.Node(xg.NodeKey("y2"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	ctx := context.Background()

	ctx, future1, err := executor.Exec(ctx, map[xg.Node]interface{}{
		x1: "X1",
		x2: "X2",
		x3: "X3",
		y1: "Y1",
		y2: "Y2",
	})

	ch1 := make(chan interface{})
	go func() {
		m := future1.Value().(map[xg.Node]Awaitable)
		ch1 <- m[ratio].Value()
		close(ch1)
	}()

	ch2 := make(chan interface{})
	go func() {
		m := future1.Value().(map[xg.Node]Awaitable)
		ch2 <- m[ratio].Value()
		close(ch2)
	}()

	exp := "ratio([sumX([X1 X2 X3]) sumY([X3 Y2 Y1])])"
	require.Equal(t, exp, <-ch1)
	require.Equal(t, exp, <-ch2)
}

func TestFlowExecPartialCalls(t *testing.T) {
	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind)
	options := Options{
		Logger: testlog{t},
	}
	executor, err := New(ref, gg, kind, options)
	require.NoError(t, err)

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	y1 := gg.Node(xg.NodeKey("y1"))
	y2 := gg.Node(xg.NodeKey("y2"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	ctx := context.Background()

	// note that each partial call gets a future
	// which is used in separate goroutines to wait
	// for the result.  We expect the futures to block
	// the same and return the same results.
	ctx, future1, err := executor.Exec(ctx, map[xg.Node]interface{}{
		x1: "X1",
		x2: "X2",
		x3: "X3",
	})
	require.NoError(t, err)

	ctx, future2, err := executor.Exec(ctx, map[xg.Node]interface{}{
		y1: "Y1",
		y2: "Y2",
	})
	require.NoError(t, err)

	ch1 := make(chan interface{})
	go func() {
		m := future1.Value().(map[xg.Node]Awaitable)
		ch1 <- m[ratio].Value()
		close(ch1)
	}()

	ch2 := make(chan interface{})
	go func() {
		m := future2.Value().(map[xg.Node]Awaitable)
		ch2 <- m[ratio].Value()
		close(ch2)
	}()

	exp := "ratio([sumX([X1 X2 X3]) sumY([X3 Y2 Y1])])"
	require.Equal(t, exp, <-ch1)
	require.Equal(t, exp, <-ch2)
}
