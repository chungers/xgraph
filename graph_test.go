package xgraph // import "github.com/orkestr8/xgraph"

import (
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

func TestAdd(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	plus := &nodeT{id: "+"}
	minus := &nodeT{id: "-"}

	g := Builder(Options{})
	require.NoError(t, g.Add(A, B, C, plus, minus))

	require.Error(t, g.Add(&nodeT{id: "A"}), "Not OK for duplicate key when struct identity fails")
	require.NoError(t, g.Add(A), "Idempotent: same node by identity")

	for _, n := range []Node{plus, minus, A, B, C} {
		require.NotNil(t, g.Node(n.NodeKey()))
	}
}

func TestAssociate(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	D := &nodeT{id: "D"}

	g := Builder(Options{})
	require.NoError(t, g.Add(A, B, C))

	require.NotNil(t, g.Node(A.NodeKey()))
	require.NotNil(t, g.Node(B.NodeKey()))
	require.NotNil(t, g.Node(C.NodeKey()))
	require.Nil(t, g.Node(D.NodeKey()))

	likes := EdgeKind(1)
	shares := EdgeKind(2)

	ev, err := g.Associate(A, likes, B, "some context", "something else")
	require.NoError(t, err)
	require.NotNil(t, g.Edge(A, likes, B))
	require.NotNil(t, ev)
	require.Equal(t, A, ev.From())
	require.Equal(t, B, ev.To())
	require.Equal(t, []interface{}{"some context", "something else"}, ev.Context())

	require.Equal(t, A, g.Edge(A, likes, B).From())
	require.Equal(t, B, g.Edge(A, likes, B).To())
	require.Equal(t, []interface{}{"some context", "something else"}, g.Edge(A, likes, B).Context())

	_, err = g.Associate(D, likes, A)
	require.Error(t, err, "Expects error because D was not added to the graph.")
	require.Nil(t, g.Edge(D, likes, A), "Expects false because C is not part of the graph.")

	_, err = g.Associate(A, likes, C)
	require.NoError(t, err, "No error because A and C are members of the graph.")
	require.NotNil(t, g.Edge(A, likes, C), "A likes C.")
	require.Equal(t, 0, len(g.Edge(A, likes, C).Context()))
	require.Nil(t, g.Edge(C, shares, A), "Shares is not an association kind between A and B.")

	require.Equal(t, 2, len(EdgeSlice(g.From(A, likes).Edges())))
	require.Equal(t, 1, len(EdgeSlice(g.To(likes, B).Edges())))
	require.Equal(t, "A", EdgeSlice(g.To(likes, B).Edges())[0].From().NodeKey())
	require.Equal(t, "B", EdgeSlice(g.To(likes, B).Edges())[0].To().NodeKey())
	require.Equal(t, 1, len(EdgeSlice(g.To(likes, C).Edges())))
	require.Equal(t, "A", EdgeSlice(g.To(likes, C).Edges())[0].From().NodeKey())
	require.Equal(t, "C", EdgeSlice(g.To(likes, C).Edges())[0].To().NodeKey())
	require.Equal(t, 0, len(EdgeSlice(g.From(B, likes).Edges())))
	require.Equal(t, 0, len(EdgeSlice(g.From(C, likes).Edges())))
	require.Equal(t, 0, len(EdgeSlice(g.To(likes, A).Edges())), "D was not added")
	require.Equal(t, 0, len(EdgeSlice(g.From(D, likes).Edges())), "D was not added")

}
