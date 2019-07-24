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
	Value() interface{}
	Error() error
	Canceled() bool
	DeadlineExceeded() bool
}

type Do func() (interface{}, error)

type future struct {
	sync.WaitGroup
	sync.RWMutex

	do    Do
	value interface{}
	err   error
	done  chan interface{}
}

func (f *future) doAsync(ctx context.Context) {
	f.Add(1)
	go func() {

		done := make(chan interface{})
		go func() {
			v, e := f.do()
			f.Yield(v, e)
			close(done)
		}()

		select {
		case <-ctx.Done():
			f.Yield(nil, ctx.Err())
			return
		case <-done:
			return
		}
	}()
}

func (f future) wait() {
	<-f.done
}

func (f *future) Canceled() bool {
	f.Wait()
	f.RLock()
	defer f.RUnlock()
	return f.err == context.Canceled
}

func (f *future) DeadlineExceeded() bool {
	f.Wait()
	f.RLock()
	defer f.RUnlock()
	return f.err == context.DeadlineExceeded
}

func (f *future) Yield(v interface{}, err error) {
	f.Lock()
	defer f.Unlock()
	if f.done != nil {
		f.value = v
		f.err = err
		close(f.done)
		f.done = nil
		f.Done()
	}
}

func (f *future) Value() interface{} {
	f.Wait()
	f.RLock()
	defer f.RUnlock()
	return f.value
}

func (f *future) Error() error {
	f.Wait()
	f.RLock()
	defer f.RUnlock()
	return f.err
}

func newFuture(do Do) *future {
	return &future{do: do, done: make(chan interface{})}
}

func Async(ctx context.Context, do Do) Awaitable {
	f := newFuture(do)
	f.doAsync(ctx)
	return f
}
