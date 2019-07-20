package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

type output struct {
	edges xg.EdgeSlice
	send  []chan<- work
}

func (output *output) dispatch(w work) {
	for _, c := range output.send {
		c <- w
	}
}
