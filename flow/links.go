package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"fmt"

	xg "github.com/orkestr8/xgraph"
)

func sendChannels(links links, keys xg.EdgeSlice) ([]chan<- work, error) {
	out := []chan<- work{}
	for _, edge := range keys {
		c, has := links[edge]
		if !has {
			return nil, fmt.Errorf("No channel allocated for edge %v", edge)
		}
		out = append(out, c)
	}
	return out, nil
}

func receiveChannels(links links, keys xg.EdgeSlice) ([]<-chan work, error) {
	out := []<-chan work{}
	for _, edge := range keys {
		c, has := links[edge]
		if !has {
			return nil, fmt.Errorf("No channel allocated for edge %v", edge)
		}
		out = append(out, c)
	}
	return out, nil
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

func (output *output) dispatch(w work) {
	for _, c := range output.send {
		c <- w
	}
}
