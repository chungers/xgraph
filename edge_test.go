package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEdgeLabel(t *testing.T) {

	var ed *edge
	ed = &edge{
		context: []interface{}{},
	}
	require.Equal(t, "", ed.label())

	ed = &edge{
		context: []interface{}{
			"foo", "bar",
		},
	}
	require.Equal(t, "foo,bar", ed.label())

	label := "my label"
	ed = &edge{
		context: []interface{}{
			func(edge Edge) string {
				return label
			},
		},
	}
	require.Equal(t, label, ed.label())

	label2 := "my label2"
	ed = &edge{
		context: []interface{}{
			func(edge Edge) string {
				return label
			},
			func(edge Edge) string {
				return label2
			},
		},
	}
	require.Equal(t, strings.Join([]string{label, label2}, ","), ed.label())
}

func TestSortEdges(t *testing.T) {

	x1 := &nodeT{id: "x1"}
	x2 := &nodeT{id: "x2"}
	x3 := &nodeT{id: "x3"}
	x4 := &nodeT{id: "x4"}
	sum := &nodeT{id: "sum"}

	input1 := EdgeKind(1)

	g := Builder(Options{})
	g.Add(x1, x2, x3, x4, sum)

	g.Associate(x1, input1, sum, 3)
	g.Associate(x2, input1, sum, 2)
	g.Associate(x3, input1, sum, 1)
	g.Associate(x4, input1, sum, 0)

	orderByContext := func(a, b Edge) bool {
		if a.To().NodeKey() != b.To().NodeKey() {
			return false
		}
		ca := a.Context()
		cb := b.Context()
		if len(ca) == 0 && len(cb) == 0 {
			return false
		}
		idx, ok := ca[0].(int)
		if ok {
			idx2, ok2 := cb[0].(int)
			if ok2 {
				return idx < idx2
			}
		}
		return false
	}

	input1s := EdgeSlice(g.To(input1, sum).Edges())

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

	input2s := EdgeSlice(g.To(input2, sum).Edges())
	SortEdges(input2s, orderByContext)

	keys = []string{}
	for i := range input1s {
		keys = append(keys, input2s[i].From().NodeKey().(string))
	}
	sort.Strings(keys) // The ordering doesn't matter in this case.
	require.Equal(t, []string{"x1", "x2", "x3", "x4"}, keys)

}
