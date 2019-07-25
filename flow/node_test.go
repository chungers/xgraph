package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"sort"
	"testing"
	//	"time"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
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
