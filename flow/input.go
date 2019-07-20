package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

type input struct {
	edges   xg.EdgeSlice
	recv    []<-chan work
	collect chan work
}

func (input *input) run() {
	for _, c := range input.recv {
		go func(cc <-chan work) {
			for {
				w, ok := <-cc
				if !ok {
					return
				}
				input.collect <- w
			}
		}(c)
	}
}

func (input *input) close() {
	close(input.collect)
}
