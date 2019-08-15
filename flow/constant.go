package flow // import "github.com/orkestr8/xgraph/flow"

type constant struct {
	value interface{}
}

func (c constant) Ch() <-chan interface{} {
	ch := make(chan interface{})
	close(ch)
	return ch
}

func (c constant) Value() interface{} {
	return c.value
}

func (c constant) Error() error {
	if err, is := c.value.(error); is {
		return err
	}
	return nil
}

func (c constant) Canceled() bool {
	return false
}

func (c constant) DeadlineExceeded() bool {
	return false
}

func (c constant) Yield(v interface{}, err error) {
	return // no-op
}

func Const(v interface{}) Awaitable {
	return &constant{value: v}
}
