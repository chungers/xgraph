package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"sync"
)

type stopper struct {
	notify map[interface{}]chan<- interface{}
	lock   sync.RWMutex
}

func (s *stopper) add(key interface{}, control chan<- interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.notify == nil {
		s.notify = map[interface{}]chan<- interface{}{}
	}
	s.notify[key] = control
}

func (s *stopper) done(key interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.notify, key)
}

func (s *stopper) waitUntil(key interface{}, other ...interface{}) {
	keys := append([]interface{}{key}, other...)
	total := 0
	for {
		for k := range keys {
			s.lock.RLock()
			if _, has := s.notify[k]; has {
				total += 1
			}
			s.lock.RUnlock()
		}
		if total == len(keys) {
			return
		}
	}
}

func (s *stopper) waitUntilDone(key interface{}, other ...interface{}) {
	keys := append([]interface{}{key}, other...)
	total := len(keys)
	for {
		for k := range keys {
			s.lock.RLock()
			if _, has := s.notify[k]; !has {
				total--
			}
			s.lock.RUnlock()
		}

		if total == 0 {
			return
		}
	}
}

func (s *stopper) stopAll() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, c := range s.notify {
		close(c)
	}
}

func (s *stopper) stop(key interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	c, has := s.notify[key]
	if has {
		close(c)
	}
}
