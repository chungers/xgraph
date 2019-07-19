package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"sync"
)

type Awaitable interface {
	Future
	Yield(v interface{}, err error)
}

type Future interface {
	Ch() <-chan interface{}
	Value() interface{}
	Error() error
	Canceled() bool
	DeadlineExceeded() bool
}

type Do func() (interface{}, error)

type future struct {
	ctx      context.Context
	do       Do
	value    interface{}
	err      error
	done     chan interface{}
	complete bool
	lock     sync.RWMutex
}

func (f *future) wait() {
	defer func() { f.complete = true }()
	for {
		select {
		case <-f.done:
			return
		case <-f.ctx.Done():
			if !f.complete {
				close(f.done)
			}
			return
		}
	}
}

func (f *future) Ch() <-chan interface{} {
	return f.done
}

func (f *future) Canceled() bool {
	return f.ctx.Err() == context.Canceled
}

func (f *future) DeadlineExceeded() bool {
	return f.ctx.Err() == context.DeadlineExceeded
}

func (f *future) Yield(v interface{}, err error) {
	f.results(v, err)
}

func (f *future) results(v interface{}, err error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	// complete is set to true exactly once, when wait completes
	// after that the value/err cannot be changed.
	if !f.complete {
		f.value = v
		f.err = err
		close(f.done)
	}
}

func (f *future) Value() interface{} {
	f.wait()
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.value
}

func (f *future) Error() error {
	f.wait()
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.err
}

func (f *future) doAsync(ctx context.Context) {
	f.ctx = ctx
	go func() {
		f.results(f.do())
		return
	}()
}

func newFuture(do Do) *future {
	return &future{do: do, done: make(chan interface{})}
}

func Async(ctx context.Context, do Do) Awaitable {
	f := newFuture(do)
	f.doAsync(ctx)
	return f
}
