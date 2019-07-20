package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"fmt"
)

type logger int

func (l logger) Log(m string, args ...interface{}) {
	if int(l) > 0 {
		fmt.Println(append([]interface{}{"INFO", m}, args...))
	}
	return
}

func (l logger) Warn(m string, args ...interface{}) {
	fmt.Println(append([]interface{}{"WARN", m}, args...))
}
