package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

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
			results <- f.Value()

			wg.Done()
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
}

func TestFutureUsageMultipleWaitersOnChan(t *testing.T) {

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

			<-f.Ch()

			require.Equal(t, "hello", f.Value())
			require.NoError(t, f.Error())

			results <- f.Value()

			wg.Done()
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

func TestFutureUsageMultipleWaiters2(t *testing.T) {

	seed := 100

	start := make(chan interface{})

	f := newFuture(func() (interface{}, error) {

		<-start

		return seed, nil
	})

	require.NotNil(t, f)

	f.doAsync(context.Background())

	c := 5000
	results := make(chan interface{}, c)

	sum := 0
	var wg sync.WaitGroup
	for i := 0; i < c; i++ {

		wg.Add(1)
		go func(i int) {

			require.Equal(t, seed, f.Value())
			require.NoError(t, f.Error())

			results <- i + f.Value().(int)

			wg.Done()
		}(i)

		sum += seed + i
	}

	close(start)

	wg.Wait()

	close(results)

	actual := 0
	for v := range results {
		actual += v.(int)
	}

	require.Equal(t, sum, actual)
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

	var wg, wg2, wg3 sync.WaitGroup
	for i := 0; i < c; i++ {

		go func(i int) {

			wg.Done()

			v := f.Value()
			e := f.Error()

			require.Equal(t, nil, v)
			require.Error(t, e)
			require.True(t, f.Canceled())

			wg2.Done()

			results <- f.Value()

			wg3.Done()
		}(i)

		wg.Add(1)
		wg2.Add(1)
		wg3.Add(1)
	}

	wg.Wait()

	// now all should be blocked
	cancel()

	wg2.Wait()

	wg3.Wait()
	close(results)

	actual := []interface{}{}
	for v := range results {
		actual = append(actual, v)
	}

	require.Equal(t, []interface{}{nil, nil, nil, nil, nil}, actual)
	require.True(t, f.Canceled())
	require.False(t, f.DeadlineExceeded())

	close(start)
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

			results <- f.Value()

			wg2.Done()
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
	fmt.Println(">>> 3")
	require.Equal(t, []interface{}{"world", "world", "world"}, actual)
	require.False(t, f.Canceled())
	require.False(t, f.DeadlineExceeded())

	close(start)

}

func testFibRecursive(i int) int {
	if i <= 1 {
		return i
	}
	return testFibRecursive(i-1) + testFibRecursive(i-2)
}

func testFibLoop(i int) int {
	f1, f2, f3 := 0, 1, 1
	for j := 2; j <= i; j++ {
		f3 = f1 + f2
		f1 = f2
		f2 = f3
	}
	return f3
}

func TestFutureUsageLongChain(t *testing.T) {

	N := 100

	start := make(chan interface{})

	ctx := context.Background()

	f0 := Async(ctx,
		func() (interface{}, error) {
			<-start
			return int(0), nil
		})

	f1 := Async(ctx,
		func() (interface{}, error) {
			return f0.Value().(int) + 1, nil
		})

	fn_1 := f1
	fn_2 := f0
	fn := fn_1

	for i := 1; i < N; i++ {
		a, b := fn_1, fn_2 // need to copy the pointer values otherwise they will change as loop variables
		fn = Async(ctx,
			func() (interface{}, error) {
				return a.Value().(int) + b.Value().(int), nil
			})

		fn_2 = fn_1
		fn_1 = fn
	}

	t0 := time.Now()
	expect := testFibLoop(N)
	dt := time.Now().Sub(t0)

	t0 = time.Now()
	close(start) // start
	actual := fn.Value()
	dt2 := time.Now().Sub(t0)

	require.Equal(t, expect, actual)
	t.Log("N=", N, "dt(fib())=", dt, "vs", "dt(chain)=", dt2, "chain is", int64(dt2)/int64(dt), "x slower")
}
