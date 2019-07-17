package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"

	xg "github.com/orkestr8/xgraph"
)

func (m flowData) hasKeys(gen func() xg.NodeSlice) bool {
	matches := 0
	test := gen()
	for _, n := range test {
		_, has := m[n]
		if has {
			matches++
		}
	}
	return len(m) == len(test)
}

// Blocks until all futures completes.
// input is an ordered slice of input edges, corresponding to the ordering of args.
func (m flowData) args(ctx context.Context, ordered xg.EdgeSlice) (args []interface{}, err error) {

	futures := []xg.Awaitable{}

	for i := range ordered {
		if future := m[ordered[i].From()]; future == nil {
			err = fmt.Errorf("%v : Missing future for %v", flowIDFrom(ctx), ordered[i])
			return
		} else {
			futures = append(futures, future)
		}
	}

	args = []interface{}{}
	// Wait for all inputs to complete computation and build args
	// for this node before proceeding with this node's computation.
	for i, future := range futures {
		// Calling the Value or Error will block until completion,
		// but can be canceled or hit deadline.
		select {
		case <-ctx.Done():
			err = fmt.Errorf("%v : Operator %v canceled while waiting for %v",
				flowIDFrom(ctx), flowOperatorFrom(ctx), ordered[i])
			return
		default:
			if err = future.Error(); err != nil {
				return
			}
			args = append(args, future.Value())
		}
	}
	return
}
