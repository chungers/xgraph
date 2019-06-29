package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"time"

	xg "github.com/orkestr8/xgraph"
)

func (fg *FlowGraph) Run(ctx context.Context,
	args map[xg.Node]interface{}) (<-chan map[xg.Node]xg.Awaitable, error) {

	callback := make(chan map[xg.Node]xg.Awaitable)
	id := flowID(time.Now().UnixNano())

	fg.Log(id, "Run with input", args)

	for k, v := range args {
		if ch, has := fg.input[k]; has {
			source := k
			arg := v
			ch <- work{
				ctx:      ctx,
				id:       id,
				from:     source,
				callback: callback,
				Awaitable: xg.Async(ctx, func() (interface{}, error) {
					fg.Log(id, source, "Exec with value=", arg)
					return arg, nil
				}),
			}
		} else {
			return nil, fmt.Errorf("not an input node %v", k)
		}
	}
	return callback, nil
}
