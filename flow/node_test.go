package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

type intNode int64

type nodeT struct {
	id         string
	custom     interface{}
	operator   func([]interface{}) (interface{}, error)
	attributes map[string]interface{}
}

func (n *nodeT) OperatorFunc() xg.OperatorFunc {
	return n.operator
}

func (n *nodeT) NodeKey() xg.NodeKey {
	return xg.NodeKey(n.id)
}

func (n *nodeT) String() string {
	return n.id
}

func (n *nodeT) Attributes() map[string]interface{} {
	return n.attributes
}

func TestNodeStartStop(t *testing.T) {
	n := &node{
		Node: &nodeT{id: "1"},
	}

	n.defaults()
	t.Log("starting")
	go n.run()

	t.Log("closing")
	n.Close() // this will cause the collection loop to end
}

func TestNodeGather(t *testing.T) {
	count := 5
	inbound := make([]chan work, count)
	typed := make([]<-chan work, count)
	for i := range inbound {
		inbound[i] = make(chan work)
		typed[i] = inbound[i]
	}
	n := &node{
		Node:    &nodeT{id: "1"},
		inbound: typed,
	}

	n.defaults()

	messages := make([]work, 1000)
	for i := range messages {
		messages[i] = work{
			id: flowID(i),
		}
	}

	start := make(chan interface{})
	sent := make(chan []int)
	collected := make(chan []int)

	go n.gather()

	// dispatch work
	go func() {
		<-start
		out := []int{}
		for i := range messages {
			s := i % len(inbound)
			inbound[s] <- messages[i]
			out = append(out, int(messages[i].id))
		}
		sent <- out
		close(sent)
	}()

	// processing
	go func() {
		m := []int{}
		for {
			w, ok := <-n.collect
			if !ok {
				break
			}
			m = append(m, int(w.id))
		}
		collected <- m
		close(collected)
	}()

	// start test
	close(start)

	expected := <-sent

	n.Close() // this will cause the collection loop to end

	received := <-collected

	sort.Ints(expected)
	sort.Ints(received)
	require.Equal(t, expected, received)

}

func TestNodeScatter(t *testing.T) {

	outbound1 := make(chan work, 1)
	outbound2 := make(chan work, 1)

	u1 := &nodeT{id: "upstream1"}
	u2 := &nodeT{id: "upstream2"}
	n := &node{
		Logger:    logger(1),
		Node:      &nodeT{id: "operator"},
		inputFrom: func() xg.NodeSlice { return []xg.Node{u1, u2} },
		outbound:  []chan<- work{outbound1, outbound2},
		then: func(args []interface{}) (interface{}, error) {
			// compute the sum
			return args[0].(int) + args[1].(int), nil
		},
	}

	n.defaults()

	go n.scatter()

	ctx := context.Background()
	a1 := Async(ctx, func() (interface{}, error) { return 100, nil })
	a2 := Async(ctx, func() (interface{}, error) { return 200, nil })

	for _, w := range []work{
		{id: 100, from: u1, ctx: ctx, Awaitable: a1},
		{id: 100, from: u2, ctx: ctx, Awaitable: a2},
	} {
		n.collect <- w
	}

	for _, c := range []chan work{outbound1, outbound2} {
		w := <-c
		require.Equal(t, n.Node, w.from)
		require.NotNil(t, w.Awaitable)
		require.Equal(t, flowID(100), w.id)
		require.Equal(t, 300, w.Value().(int))
		require.Nil(t, w.Error())
	}

	n.Close()
}

func TestNodeApply(t *testing.T) {

	c := 1000
	args := []interface{}{1, 2}

	n := &node{
		sem: semaphore.NewWeighted(2),
		then: func(args []interface{}) (interface{}, error) {
			// compute the sum
			return args[0].(int) + args[1].(int), nil
		},
	}

	start := make(chan interface{})

	results := map[int]chan int{}
	keys := make([]int, c)
	for i := 0; i < c; i++ {
		keys[i] = i
		results[i] = make(chan int, 1)
	}

	ctx := context.Background()

	wg := sync.WaitGroup{}
	for i := range results {
		wg.Add(1)
		go func(i int) {
			<-start
			v, err := n.apply(ctx, args)
			require.NoError(t, err)
			vv, _ := n.then(args)
			require.Equal(t, vv, v.(int))
			results[i] <- i // send the id to verify execution
			wg.Done()
		}(i)
	}

	close(start)

	wg.Wait()

	for i := range keys {
		require.NotNil(t, results[i])
		if <-results[i] == i {
			close(results[i])
			delete(results, i)
		}
	}

	require.Equal(t, 0, len(results))
}

func TestNodeApplyCancel(t *testing.T) {

	c := 1000
	args := []interface{}{1, 2}

	n := &node{
		sem: semaphore.NewWeighted(0),
		then: func(args []interface{}) (interface{}, error) {
			// compute the sum
			return args[0].(int) + args[1].(int), nil
		},
	}

	start := make(chan interface{})

	results := map[int]chan int{}
	keys := make([]int, c)
	for i := 0; i < c; i++ {
		keys[i] = i
		results[i] = make(chan int, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	for i := range results {
		wg.Add(1)
		go func(i int) {
			<-start

			_, err := n.apply(ctx, args)
			require.Error(t, err)
			results[i] <- i // send the id to verify execution

			wg.Done()
		}(i)
	}

	close(start)

	cancel()

	wg.Wait()

	for i := range keys {
		require.NotNil(t, results[i])
		if <-results[i] == i {
			close(results[i])
			delete(results, i)
		}
	}

	require.Equal(t, 0, len(results))
}

func TestNodeApplyAsync(t *testing.T) {

	c := 20

	inputs := 1000

	ctx := context.Background()
	g := map[xg.Node]Awaitable{}
	for i := 0; i < inputs; i++ {
		g[&nodeT{id: fmt.Sprintf("%v", i)}] = Async(ctx, func() (interface{}, error) { return 1, nil })
	}

	n := &node{
		sem: semaphore.NewWeighted(1),
		inputFrom: func() xg.NodeSlice {
			s := xg.NodeSlice{}
			for k := range g {
				s = append(s, k)
			}
			return s
		},
		then: func(args []interface{}) (interface{}, error) {
			// compute the sum
			sum := 0
			for i := range args {
				sum += args[i].(int)
			}
			return sum, nil
		},
	}

	start := make(chan interface{})

	results := map[int]chan int{}
	keys := make([]int, c)
	for i := 0; i < c; i++ {
		keys[i] = i
		results[i] = make(chan int, 1)
	}

	wg := sync.WaitGroup{}
	for i := range results {
		wg.Add(1)
		go func(i int) {

			<-start

			future := n.applyAsync(ctx, g)

			require.NoError(t, future.Error())
			require.Equal(t, 1*inputs, future.Value()) // just a sum of 1 * input times
			results[i] <- i                            // send the id to verify execution

			wg.Done()
		}(i)
	}

	close(start)

	wg.Wait()

	for i := range keys {
		require.NotNil(t, results[i])
		if <-results[i] == i {
			close(results[i])
			delete(results, i)
		}
	}

	require.Equal(t, 0, len(results))
}
