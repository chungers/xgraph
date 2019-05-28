package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFutureUsageMultipleWaiters(t *testing.T) {

	start := make(chan interface{})

	ctx := context.Background()
	f := Async(ctx,
		func() (interface{}, error) {

			<-start

			return "hello", nil
		})

	require.NotNil(t, f)

	c := 5
	results := make(chan interface{}, c)

	var wg sync.WaitGroup
	for i := 0; i < c; i++ {

		wg.Add(1)
		go func(i int) {

			require.Equal(t, "hello", f.Value())
			require.NoError(t, f.Error())

			wg.Done()

			results <- f.Value()
		}(i)
	}

	close(start)
	wg.Wait()

	close(results)

	check := []interface{}{}
	for i := 0; i < c; i++ {
		require.Equal(t, nil, f.Error())
		require.Equal(t, "hello", f.Value())
		check = append(check, f.Value())
	}

	actual := []interface{}{}
	for v := range results {
		actual = append(actual, v)
	}

	require.Equal(t, check, actual)
	require.True(t, f.(*future).complete)
}

func TestFutureUsageMultipleWaitersCancellation(t *testing.T) {

	start := make(chan interface{})

	ctx, cancel := context.WithCancel(context.Background())

	f := Async(ctx,
		func() (interface{}, error) {

			<-start

			return "hello", nil
		})

	require.NotNil(t, f)

	c := 5
	results := make(chan interface{}, c)

	var wg, wg2 sync.WaitGroup
	for i := 0; i < c; i++ {

		go func(i int) {

			wg.Done()

			require.Equal(t, nil, f.Value())
			require.NoError(t, f.Error())

			wg2.Done()

			results <- f.Value()

		}(i)

		wg.Add(1)
		wg2.Add(1)
	}

	wg.Wait()

	// now all should be blocked
	cancel()

	wg2.Wait()
	close(results)

	actual := []interface{}{}
	for v := range results {
		actual = append(actual, v)
	}

	require.Equal(t, []interface{}{nil, nil, nil, nil, nil}, actual)
	require.True(t, f.Canceled())
	require.False(t, f.DeadlineExceeded())

	close(start)
	require.True(t, f.(*future).complete)
}

func TestFutureUsageMultipleWaitersInjectValues(t *testing.T) {

	start := make(chan interface{})

	ctx := context.Background()

	f := Async(ctx,
		func() (interface{}, error) {

			<-start

			return "hello", nil
		})

	require.NotNil(t, f)

	c := 3
	results := make(chan interface{}, c)

	var wg, wg2 sync.WaitGroup
	for i := 0; i < c; i++ {

		go func(i int) {

			wg.Done()

			require.Equal(t, "world", f.Value())
			require.NoError(t, f.Error())

			wg2.Done()

			results <- f.Value()

		}(i)

		wg.Add(1)
		wg2.Add(1)
	}

	wg.Wait()

	f.(Awaitable).Yield("world", nil)

	wg2.Wait()
	close(results)

	actual := []interface{}{}
	for v := range results {
		actual = append(actual, v)
	}

	require.Equal(t, []interface{}{"world", "world", "world"}, actual)
	require.False(t, f.Canceled())
	require.False(t, f.DeadlineExceeded())

	close(start)
	require.True(t, f.(*future).complete)

}
