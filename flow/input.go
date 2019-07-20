package flow // import "github.com/orkestr8/xgraph/flow"

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
