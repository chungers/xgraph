package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func TestGonumGraph(t *testing.T) {

	g := simple.NewDirectedGraph()

	// AddNode must be called right after NewNode to ensure the ID is properly assigned and registered
	// in the graph, or we'd get ID collision panic.
	a := g.NewNode()
	require.Nil(t, g.Node(a.ID()))
	g.AddNode(a)
	require.NotNil(t, g.Node(a.ID()))

	b := g.NewNode()
	g.AddNode(b)

	aLikesB := g.NewEdge(a, b)
	require.Nil(t, g.Edge(a.ID(), b.ID()))
	g.SetEdge(aLikesB)

	cycle := topo.DirectedCyclesIn(g)
	require.Equal(t, 0, len(cycle))

	// Calling ReversedEdge doesn't actually reverses the edge in the graph.
	reversed := aLikesB.ReversedEdge()
	require.Nil(t, g.Edge(b.ID(), a.ID()))
	require.NotNil(t, g.Edge(a.ID(), b.ID()))

	// Now an edge exists.  For this DAG we have a loop now.
	g.SetEdge(reversed)
	require.NotNil(t, g.Edge(b.ID(), a.ID()))
	require.NotNil(t, g.Edge(a.ID(), b.ID()))

	_, err := topo.SortStabilized(g, nil)
	require.Error(t, err)

	cycle = topo.DirectedCyclesIn(g)
	require.Equal(t, 1, len(cycle))
	t.Log(cycle)

	c := g.NewNode()
	g.AddNode(c)
	g.SetEdge(g.NewEdge(a, c))
	g.SetEdge(g.NewEdge(c, a))
	cycle = topo.DirectedCyclesIn(g)
	require.Equal(t, 2, len(cycle))
	t.Log(cycle)
}

type nodeT struct {
	id string
}

func (n *nodeT) Key() NodeKey {
	return NodeKey(n.id)
}

func (n *nodeT) String() string {
	return n.id
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
	D := &nodeT{id: "D"}

	g := New(Options{})
	require.NoError(t, g.Add(A, B, C))

	require.True(t, g.Has(A))
	require.True(t, g.Has(B))
	require.True(t, g.Has(C))
	require.False(t, g.Has(D))

	likes := EdgeKind(1)
	shares := EdgeKind(2)

	_, err := g.Associate(A, likes, B)
	require.NoError(t, err)
	require.True(t, g.Edge(A, likes, B))

	_, err = g.Associate(D, likes, A)
	require.Error(t, err, "Expects error because D was not added to the graph.")
	require.False(t, g.Edge(D, likes, A), "Expects false because C is not part of the graph.")

	_, err = g.Associate(A, likes, C)
	require.NoError(t, err, "No error because A and C are members of the graph.")
	require.True(t, g.Edge(A, likes, C), "A likes C.")
	require.False(t, g.Edge(C, shares, A), "Shares is not an association kind between A and B.")

}

func TestPaths(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	D := &nodeT{id: "D"}

	g := New(Options{})
	require.NoError(t, g.Add(A, B, C, D))

	refs := EdgeKind(1)
	likes := EdgeKind(2)

	g.Associate(A, refs, B)
	g.Associate(B, refs, C)
	g.Associate(C, refs, D)

	gn, err := g.(*graph).toGonum(refs, Path{A, B, C, D})
	require.NoError(t, err)
	require.Equal(t, 4, len(gn))

	nn, err := g.(*graph).fromGonum(refs, gn)
	require.NoError(t, err)
	require.Equal(t, Path{A, B, C, D}, nn)

	gn2, err := g.(*graph).toGonum(likes, Path{A, B})
	require.NoError(t, err)
	require.Equal(t, 0, len(gn2))

	nn2, err := g.(*graph).fromGonum(likes, gn)
	require.NoError(t, err)
	require.Equal(t, 0, len(nn2))

	cycles, err := DirectedCycles(g, refs)
	require.NoError(t, err)
	require.Equal(t, 0, len(cycles))
}

func TestDirectedCycles(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	D := &nodeT{id: "D"}

	g := New(Options{})
	require.NoError(t, g.Add(A, B, C, D))

	refs := EdgeKind(1)

	g.Associate(A, refs, B)
	g.Associate(B, refs, C)
	g.Associate(C, refs, D)
	g.Associate(D, refs, A)

	cycles, err := DirectedCycles(g, refs)
	require.NoError(t, err)
	require.Equal(t, 1, len(cycles))
	require.Equal(t, Path{A, B, C, D, A}, cycles[0])

	imports := EdgeKind(2)
	g.Associate(A, imports, B)
	g.Associate(B, imports, A)
	g.Associate(B, imports, C)
	g.Associate(C, imports, A)

	cycles, err = DirectedCycles(g, imports)
	require.NoError(t, err)

	for i := range cycles {
		t.Log(g.(*graph).toGonum(imports, cycles[i]))
	}
	require.Equal(t, 2, len(cycles))
	require.Equal(t, Path{A, B, C, A}, cycles[0])
	require.Equal(t, Path{A, B, A}, cycles[1])
}

func TestDirectedSort(t *testing.T) {

	g := New(Options{})
	next := EdgeKind(1)
	prev := EdgeKind(2)

	m := 4
	var last Node
	for i := 0; i < m; i++ {
		this := &nodeT{id: fmt.Sprintf("N%v", i)}
		g.Add(this)

		if last != nil {
			g.Associate(last, next, this)
			g.Associate(this, prev, last)
		}
		last = this
	}

	forward, err := DirectedSort(g, next)
	require.NoError(t, err)

	backward, err := DirectedSort(g, prev)
	require.NoError(t, err)

	for i := range forward {
		require.Equal(t, forward[i], backward[len(backward)-i-1])
	}

	require.Equal(t, Reverse(forward), backward)

	// add more nodes for the next relation
	for i := m; i < 2*m; i++ {
		this := &nodeT{id: fmt.Sprintf("N%v", i)}
		g.Add(this)

		if last != nil {
			g.Associate(this, next, last)
		}
		last = this
	}

	forward, err = DirectedSort(g, next)
	require.NoError(t, err)

	// The last node should be the mid-point
	// 0 -> 1 -> 2 -> 3 <- 4 <- 5 <- 6 <- 7 for m = 4
	require.Equal(t, fmt.Sprintf("N%v", m-1), forward[len(forward)-1].(*nodeT).id)

	t.Log(forward)
}
