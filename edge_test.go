package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortEdges(t *testing.T) {

	x1 := &nodeT{id: "x1"}
	x2 := &nodeT{id: "x2"}
	x3 := &nodeT{id: "x3"}
	x4 := &nodeT{id: "x4"}
	sum := &nodeT{id: "sum"}

	input1 := EdgeKind(1)

	g := Builder(Options{})
	g.Add(x1, x2, x3, x4, sum)

	g.Associate(x1, input1, sum, Attribute{Key: "order", Value: 3})
	g.Associate(x2, input1, sum, Attribute{Key: "order", Value: 2})
	g.Associate(x3, input1, sum, Attribute{Key: "order", Value: 1})
	g.Associate(x4, input1, sum, Attribute{Key: "order", Value: 0})

	orderByContext := func(a, b Edge) bool {
		if a.To().NodeKey() != b.To().NodeKey() {
			return false
		}
		ca := a.Attributes()
		cb := b.Attributes()
		if len(ca) == 0 && len(cb) == 0 {
			return false
		}
		idx, ok := ca["order"].(int)
		if ok {
			idx2, ok2 := cb["order"].(int)
			if ok2 {
				return idx < idx2
			}
		}
		return false
	}

	input1s := g.To(input1, sum).Edges().Slice()

	t.Log(input1s)

	SortEdges(input1s, orderByContext)

	t.Log("sorted=", input1s)

	keys := []string{}
	for i := range input1s {
		keys = append(keys, input1s[i].From().NodeKey().(string))
	}
	require.Equal(t, []string{"x4", "x3", "x2", "x1"}, keys)

	input2 := EdgeKind(2)
	g.Associate(x1, input2, sum)
	g.Associate(x2, input2, sum)
	g.Associate(x3, input2, sum)
	g.Associate(x4, input2, sum)

	input2s := g.To(input2, sum).Edges().Slice()
	SortEdges(input2s, orderByContext)

	keys = []string{}
	for i := range input1s {
		keys = append(keys, input2s[i].From().NodeKey().(string))
	}
	sort.Strings(keys) // The ordering doesn't matter in this case.
	require.Equal(t, []string{"x1", "x2", "x3", "x4"}, keys)

}
