package xgraph // import "github.com/orkestr8/xgraph"

import (
	"context"
	"sync"
)

type Async interface {
	Value() interface{}
	Error() error
}

type Do func() (interface{}, error)

type future struct {
	ctx   context.Context
	value interface{}
	err   error
	done  chan interface{}
	lock  sync.RWMutex
}

func (f *future) wait() {
	for {
		select {
		case _, open := <-f.done:
			if !open {
				return
			}
		case <-f.ctx.Done():
			return
		}
	}
}

func (f *future) results(v interface{}, err error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.value = v
	f.err = err
	close(f.done)
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

func Future(ctx context.Context, do Do) Async {

	f := &future{
		ctx:  ctx,
		done: make(chan interface{}),
	}

	go func() {
		f.results(do())
		return
	}()
	return f
}
