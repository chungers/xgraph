package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"time"

	xg "github.com/orkestr8/xgraph"
)

func (fg *FlowGraph) Run(ctx context.Context,
	args map[xg.Node]interface{}) (<-chan gather, error) {

	callback := make(chan gather)
	id := flowID(time.Now().UnixNano())

	fg.Log("Start flow run", "id", id, "args", args)

	for k, v := range args {
		if ch, has := fg.input[k]; has {
			source := k
			arg := v
			ch <- work{
				ctx:      ctx,
				id:       id,
				from:     source,
				callback: callback,
				Awaitable: Async(ctx, func() (interface{}, error) {
					return arg, nil
				}),
			}
		} else {
			return nil, fmt.Errorf("not an input node %v", k)
		}
	}
	return callback, nil
}
