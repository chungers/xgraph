package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"sync"
)

type memoized struct {
	value interface{}
	err   error
}

type inline struct {
	op   func() (interface{}, error)
	memo *memoized
	lock sync.RWMutex
}

func (c inline) Ch() <-chan interface{} {
	ch := make(chan interface{})
	close(ch)
	return ch
}

func (c *inline) memoized() *memoized {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.memo
}

func (c *inline) memoize(v interface{}, err error) *memoized {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.memo = &memoized{
		value: v,
		err:   err,
	}
	return c.memo
}

func (c *inline) Value() interface{} {
	mem := c.memoized()
	if mem != nil {
		return mem.value
	}
	return c.memoize(c.op()).value
}

func (c *inline) Error() error {
	mem := c.memoized()
	if mem != nil {
		return mem.err
	}
	return c.memoize(c.op()).err
}

func (c inline) Canceled() bool {
	return false
}

func (c inline) DeadlineExceeded() bool {
	return false
}

func (c inline) Yield(v interface{}, err error) {
	return // no-op
}

func Inline(f func() (interface{}, error)) Awaitable {
	return &inline{op: f}
}
