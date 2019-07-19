package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

func TestGatherHasKeys(t *testing.T) {

	n1 := &nodeT{id: "1"}
	n2 := &nodeT{id: "2"}
	n3 := &nodeT{id: "3"}
	n4 := &nodeT{id: "4"}
	n5 := &nodeT{id: "5"}
	n6 := &nodeT{id: "6"}

	gt := gather{n1: nil, n2: nil, n3: nil, n4: nil, n5: nil}
	require.True(t, gt.hasKeys(func() xg.NodeSlice { return nil }))

	require.False(t, gt.hasKeys(
		func() xg.NodeSlice {
			return xg.NodeSlice{n6, n2}
		}))
	require.True(t, gt.hasKeys(
		func() xg.NodeSlice {
			return xg.NodeSlice{n1, n2, n3}
		}))
}

func TestOrderFutures(t *testing.T) {
	n1 := &nodeT{id: "1"}
	n2 := &nodeT{id: "2"}
	n3 := &nodeT{id: "3"}
	n4 := &nodeT{id: "4"}
	n5 := &nodeT{id: "5"}

	ctx := context.Background()
	done := make(chan interface{})

	a1 := xg.Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})
	a2 := xg.Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})
	a3 := xg.Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})
	a4 := xg.Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})

	gs := gather{n1: a1, n2: a2, n3: a3, n4: a4}
	f, err := gs.futuresForNodes(ctx, xg.EdgeSlice{}.FromNodes)
	require.NoError(t, err)
	require.Equal(t, 0, len(f))

	f, err = gs.futuresForNodes(ctx,
		func() xg.NodeSlice {
			return xg.NodeSlice{n1, n2}
		})
	require.NoError(t, err)
	require.Equal(t, []xg.Awaitable{a1, a2}, f)

	f, err = gs.futuresForNodes(ctx,
		func() xg.NodeSlice {
			return xg.NodeSlice{n1, n5}
		})
	require.Error(t, err)
}
