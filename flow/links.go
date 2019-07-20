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
