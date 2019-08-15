package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"

	xg "github.com/orkestr8/xgraph"
)

type gather map[xg.Node]Awaitable

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

func (m gather) futuresForNodes(ctx context.Context,
	gen func() xg.NodeSlice) (ordered xg.NodeSlice, futures []Future, err error) {

	futures = []Future{}
	ordered = gen()
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

// Wait for the futures to complete.  Note the futures are themselves cancellable, if the
// contexts given cancels.
func waitFor(futures []Future) ([]interface{}, error) {
	out := []interface{}{}
	for _, f := range futures {
		if err := f.Error(); err != nil {
			return nil, err
		} else {
			out = append(out, f.Value())
		}
	}
	return out, nil
}
