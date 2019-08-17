package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

func TestFlowNew(t *testing.T) {
	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind)
	options := Options{}
	executor, err := NewExecutor(ref, gg, kind, options)
	require.NoError(t, err)

	require.NoError(t, executor.Close())
}

func TestFlowExecFullAsync(t *testing.T) {
	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind)
	options := Options{}
	executor, err := NewExecutor(ref, gg, kind, options)
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

func TestFlowExecFullInline(t *testing.T) {
	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind, true)
	options := Options{}
	executor, err := NewExecutor(ref, gg, kind, options)
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
	options := Options{}
	executor, err := NewExecutor(ref, gg, kind, options)
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

func BenchmarkCompile(b *testing.B) {

	for i := 0; i < b.N; i++ {

		ref := GraphRef("test")
		kind := xg.EdgeKind(1)
		gg := testBuildGraph(kind)
		options := Options{}

		executor, err := NewExecutor(ref, gg, kind, options)
		require.NoError(b, err)
		require.NoError(b, executor.Close())
	}
}

func BenchmarkExecWithConstsAsync(b *testing.B) {
	benchmarkExecWithConsts(b, false)
}

func BenchmarkExecWithConstsInline(b *testing.B) {
	benchmarkExecWithConsts(b, true)
}

func benchmarkExecWithConsts(b *testing.B, inline bool) {

	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind, inline)
	options := Options{}

	executor, err := NewExecutor(ref, gg, kind, options)
	require.NoError(b, err)

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	y1 := gg.Node(xg.NodeKey("y1"))
	y2 := gg.Node(xg.NodeKey("y2"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	exp := "ratio([sumX([X1 X2 X3]) sumY([X3 Y2 Y1])])"

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, future, _ := executor.Exec(ctx, map[xg.Node]interface{}{
			x1: "X1",
			x2: "X2",
			x3: "X3",
			y1: "Y1",
			y2: "Y2",
		})

		m := future.Value().(map[xg.Node]Awaitable)
		require.Equal(b, exp, m[ratio].Value())
	}

	require.NoError(b, executor.Close())
}

func BenchmarkExecWithAwaitablesAsync(b *testing.B) {
	benchmarkExecWithAwaitables(b, false)
}

func BenchmarkExecWithAwaitablesInline(b *testing.B) {
	benchmarkExecWithAwaitables(b, true)
}

func benchmarkExecWithAwaitables(b *testing.B, inline bool) {

	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind, inline)
	options := Options{}

	executor, err := NewExecutor(ref, gg, kind, options)
	require.NoError(b, err)

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	y1 := gg.Node(xg.NodeKey("y1"))
	y2 := gg.Node(xg.NodeKey("y2"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	exp := "ratio([sumX([X1 X2 X3]) sumY([X3 Y2 Y1])])"

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, future, _ := executor.ExecAwaitables(ctx, map[xg.Node]Awaitable{
			x1: Async(ctx, func() (interface{}, error) { return "X1", nil }),
			x2: Async(ctx, func() (interface{}, error) { return "X2", nil }),
			x3: Async(ctx, func() (interface{}, error) { return "X3", nil }),
			y1: Async(ctx, func() (interface{}, error) { return "Y1", nil }),
			y2: Async(ctx, func() (interface{}, error) { return "Y2", nil }),
		})
		m := future.Value().(map[xg.Node]Awaitable)
		require.Equal(b, exp, m[ratio].Value())
	}

	require.NoError(b, executor.Close())
}

func TestFlowExecFullConcurrentAsync(t *testing.T) {
	testFlowExecFullConcurrent(t, false)
}

func TestFlowExecFullConcurrentInline(t *testing.T) {
	testFlowExecFullConcurrent(t, true)
}

func testFlowExecFullConcurrent(t *testing.T, inline bool) {
	ref := GraphRef("test")
	kind := xg.EdgeKind(1)
	gg := testBuildGraph(kind, inline)
	options := Options{}
	executor, err := NewExecutor(ref, gg, kind, options)
	require.NoError(t, err)

	x1 := gg.Node(xg.NodeKey("x1"))
	x2 := gg.Node(xg.NodeKey("x2"))
	x3 := gg.Node(xg.NodeKey("x3"))
	y1 := gg.Node(xg.NodeKey("y1"))
	y2 := gg.Node(xg.NodeKey("y2"))
	ratio := gg.Node(xg.NodeKey("ratio"))

	client := func(t *testing.T, i int, wg *sync.WaitGroup, input map[xg.Node]interface{}) {
		exp := "ratio([sumX([x1 x2 x3]) sumY([x3 y2 y1])])" // template
		// substitute real input
		for n, v := range input {
			exp = strings.ReplaceAll(exp, n.NodeKey().(string), fmt.Sprintf("%v", v))
		}

		_, future, err := executor.Exec(context.Background(), input)
		require.NoError(t, err)
		require.Equal(t, exp, future.Value().(map[xg.Node]Awaitable)[ratio].Value())
		wg.Done()
	}

	rand.Seed(42)

	wg := &sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go client(t, i, wg, map[xg.Node]interface{}{
			x1: rand.Int(),
			x2: rand.Int(),
			x3: rand.Int(),
			y1: rand.Int(),
			y2: rand.Int(),
		})
	}

	wg.Wait()
}
