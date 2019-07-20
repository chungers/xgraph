package flow // import "github.com/orkestr8/xgraph/flow"

func (output *output) dispatch(w work) {
	for _, c := range output.send {
		c <- w
	}
}
