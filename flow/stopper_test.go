package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStopperAddDone(t *testing.T) {
	s := stopper{}

	count := 10
	wg := sync.WaitGroup{}
	chs := make([]chan interface{}, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(k int) {
			ch := make(chan interface{})
			chs[k] = ch
			s.add(k, ch)
			wg.Done()
		}(i)
	}

	wg.Wait()
	require.Equal(t, count, len(s.notify))

	wg = sync.WaitGroup{}
	for i := 0; i < count/2; i++ {
		wg.Add(1)
		go func(k int) {
			s.done(k)
			wg.Done()
		}(i)
	}
	wg.Wait()
	require.Equal(t, count/2, len(s.notify))
}

func TestStopperWaitUntil(t *testing.T) {

	started := make(chan bool, 1)

	count := 10
	services := map[int]chan interface{}{}

	for i := 0; i < count; i++ {
		services[i] = make(chan interface{})
	}

	s := stopper{}
	go func() {

		keys := []interface{}{}
		for k := range services {
			keys = append(keys, k)
		}

		s.waitUntil(keys[0], keys[1:]...)

		started <- true
		close(started)
	}()

	wg := sync.WaitGroup{}
	chs := make([]chan interface{}, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(k int) {
			ch := make(chan interface{})
			chs[k] = ch
			s.add(k, ch)
			wg.Done()
		}(i)
	}
	wg.Wait()

	require.True(t, <-started)
}

func TestStopperWaitUntilAllDone(t *testing.T) {

	count := 10
	services := map[int]chan interface{}{}

	for i := 0; i < count; i++ {
		services[i] = make(chan interface{})
	}

	s := stopper{}

	// register services
	chs := make([]chan interface{}, count)
	for i := 0; i < count; i++ {
		go func(k int) {
			ch := make(chan interface{})
			chs[k] = ch
			s.add(k, ch)

			// block here until someone closes the ch
			<-ch

			// unregisters the service
			s.done(k)

		}(i)
	}

	// wait until they are all registered/ running
	keys := []interface{}{}
	for k := range services {
		keys = append(keys, k)
	}

	s.waitUntil(keys[0], keys[1:]...)

	// Wait to avoid a race
	time.Sleep(500 * time.Millisecond)

	finished := make(chan bool, 1)
	go func() {

		s.waitUntilDone(keys[0], keys[1:]...)

		finished <- true

		return
	}()

	// now start done
	for i := 0; i < count/2; i++ {
		go func(k int) {
			s.stop(services[k])

			return
		}(i)
	}

	// finish all
	s.stopAll()

	require.True(t, <-finished)
}
