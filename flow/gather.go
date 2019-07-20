package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"reflect"

	xg "github.com/orkestr8/xgraph"
)

type gather map[xg.Node]xg.Awaitable

func (m gather) hasKeys(gen func() xg.NodeSlice) bool {
	matches := 0
	test := gen()
	for _, n := range test {
		_, has := m[n]
		if has {
			matches++
		}
	}
	return matches == len(test)
}

func (m gather) futuresForNodes(ctx context.Context, gen func() xg.NodeSlice) (futures []xg.Future, err error) {
	futures = []xg.Future{}
	ordered := gen()
	for i := range ordered {
		if future := m[ordered[i]]; future == nil {
			err = fmt.Errorf("%v : Missing future for %v", flowIDFrom(ctx), ordered[i])
			return
		} else {
			futures = append(futures, future)
		}
	}
	return
}

func waitFor(ctx context.Context, futures []xg.Future) ([]interface{}, error) {

	args := make([]interface{}, len(futures))

	cases := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ctx.Done()),
		},
	}
	for i := range futures {
		cases = append(cases,
			reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(futures[i].Ch()),
			})
	}

	for i := 0; i < len(futures); i++ {
		// The channel used by the future isn't for passing values. It's only for signaling.
		// Therefore, we don't need to get the value and should access the value of the future
		// via the Value() and Error() methods.
		index, _, ok := reflect.Select(cases)
		if !ok {
			cases[index].Chan = reflect.ValueOf(nil)
			if index == 0 {
				return nil, ctx.Err()
			}
		}
		if err := futures[index-1].Error(); err != nil {
			return nil, err
		}
		args[index-1] = futures[index-1].Value()
	}

	return args, nil
}
