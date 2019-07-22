package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"sort"
	"testing"

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
