package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFutureUsage(t *testing.T) {

	start := make(chan interface{})

	ctx := context.Background()
	f := Future(ctx,
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
}
