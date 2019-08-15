package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"testing"
	"time"

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

	ctx := context.Background()
	done := make(chan interface{})

	a1 := Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})
	a2 := Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})
	a3 := Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})
	a4 := Async(ctx, func() (interface{}, error) {
		<-done
		return nil, nil
	})

	gs := gather{n1: a1, n2: a2, n3: a3, n4: a4}
	_, f, err := gs.futuresForNodes(ctx, xg.EdgeSlice{}.FromNodes)
	require.NoError(t, err)
	require.Equal(t, 0, len(f))

	_, f, err = gs.futuresForNodes(ctx,
		func() xg.NodeSlice {
			return xg.NodeSlice{n1, n2}
		})
	require.NoError(t, err)
	require.Equal(t, []Future{a1, a2}, f)

	_, f, err = gs.futuresForNodes(ctx,
		func() xg.NodeSlice {
			return xg.NodeSlice{n1, &nodeT{id: "?"}}
		})
	require.Error(t, err)
}

func TestWaitForNormal(t *testing.T) {

	c := []chan interface{}{}
	f := []Future{}

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		cc := make(chan interface{})
		aa := Async(ctx, func() (interface{}, error) {
			v := <-cc
			return v, nil
		})
		c = append(c, cc)
		f = append(f, aa)
	}

	start := make(chan interface{})
	go func() {
		<-start
		for i := range c {
			c[i] <- i
			close(c[i])
		}
	}()

	result := make(chan []interface{})
	go func() {
		args, err := waitFor(f)
		result <- []interface{}{args, err}
	}()

	close(start)
	r := <-result

	require.Nil(t, r[1])
	for i, v := range r[0].([]interface{}) {
		require.Equal(t, i, v)
	}

	close(result)
}

func TestWaitForContextCancel(t *testing.T) {

	block := make(chan interface{})
	f := []Future{}

	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < 5; i++ {
		aa := Async(ctx, func() (interface{}, error) {
			<-block
			return i, nil
		})
		f = append(f, aa)
	}

	done := make(chan interface{})
	go func() {
		defer close(done)
		args, err := waitFor(f)
		require.Error(t, err)
		require.Nil(t, args)
	}()

	time.Sleep(5 * time.Millisecond)

	cancel()

	<-done
}

func TestWaitForContextAsyncError(t *testing.T) {

	c := []chan interface{}{}
	f := []Future{}
	verr := fmt.Errorf("boom")
	v := []interface{}{5, 4, 3, 2, 1, verr}
	ctx, _ := context.WithCancel(context.Background())

	for i := 0; i < len(v); i++ {
		cc := make(chan interface{})
		aa := Async(ctx, func() (interface{}, error) {
			x := <-cc
			if x, is := x.(error); is {
				return nil, x
			}
			return x, nil
		})
		c = append(c, cc)
		f = append(f, aa)
	}

	done := make(chan interface{})
	go func() {
		defer close(done)
		args, err := waitFor(f)
		require.Error(t, err)
		require.Equal(t, verr, err)
		require.Nil(t, args)
	}()

	// send the values
	for i := range v {
		c[i] <- v[i]
		close(c[i])
	}

	<-done
}

func TestWaitForContextTimeout(t *testing.T) {

	c := []chan interface{}{}
	f := []Future{}
	v := []interface{}{5, 4, 3, 2, 1}
	ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

	for i := 0; i < len(v); i++ {
		cc := make(chan interface{})
		aa := Async(ctx, func() (interface{}, error) {
			x := <-cc
			if x, is := x.(error); is {
				return nil, x
			}
			return x, nil
		})
		c = append(c, cc)
		f = append(f, aa)
	}

	done := make(chan interface{})
	go func() {
		defer close(done)
		args, err := waitFor(f)
		require.Error(t, err)
		require.Nil(t, args)
	}()

	time.Sleep(1 * time.Second) // this comes AFTER the timeout of 0.5 second

	// send the values
	for i := range v {
		c[i] <- v[i]
		close(c[i])
	}

	<-done
}
