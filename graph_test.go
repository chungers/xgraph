package xgraph // import "github.com/orkestr8/xgraph"

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type nodeT struct {
	id string
}

func (n *nodeT) Key() NodeKey {
	return []byte(n.id)
}

func TestAdd(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	plus := &nodeT{id: "+"}
	minus := &nodeT{id: "-"}

	g := New(Options{})
	require.NoError(t, g.Add(A, B, C, plus, minus))

	require.NoError(t, g.Add(A), "Idempotent: same node by identity")
	require.NoError(t, g.Add(&nodeT{id: "A"}), "OK for duplicate key when struct identity fails")

	for _, n := range []Node{plus, minus, A, B, C} {
		require.True(t, g.Has(n))
	}
}

func TestAssociate(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}

	g := New(Options{})
	require.NoError(t, g.Add(A, B))

	require.True(t, g.Has(A))
	require.True(t, g.Has(B))
	require.False(t, g.Has(C))

	likes := EdgeKind(1)

	_, err := g.Associate(likes, C, A)
	require.Error(t, err)
}
