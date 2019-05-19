package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaths(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	D := &nodeT{id: "D"}

	g := Builder(Options{})
	require.NoError(t, g.Add(A, B, C, D))

	refs := EdgeKind(1)

	g.Associate(A, refs, B)
	g.Associate(B, refs, C)
	g.Associate(C, refs, D)

	gn := g.(*graph).directed[refs].gonum(A, B, C, D)
	require.Equal(t, 4, len(gn))
	for i := range gn {
		require.NotNil(t, gn[i])
	}

	nn := g.(*graph).directed[refs].xgraph(gn[0], gn[1:]...)
	require.Equal(t, Path{A, B, C, D}, Path(nn))

	cycles, err := DirectedCycles(g, refs)
	require.NoError(t, err)
	require.Equal(t, 0, len(cycles))
}

func TestScopeDirected(t *testing.T) {

	err := scopeDirected(nil, EdgeKind(1),
		func(dg *directed) error {
			return nil
		})
	require.Error(t, err)

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}

	g := Builder(Options{})
	require.NoError(t, g.Add(A, B))
	refs := EdgeKind(1)
	g.Associate(A, refs, B)

	count := 0
	err = scopeDirected(g, EdgeKind(1),
		func(dg *directed) error {
			require.NotNil(t, dg)

			count = len(dg.edges)
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestDirectedCycles(t *testing.T) {

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	D := &nodeT{id: "D"}

	g := Builder(Options{})
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
	g.Associate(C, imports, B)

	cycles, err = DirectedCycles(g, imports)
	require.NoError(t, err)

	for i := range cycles {
		t.Log(g.(*graph).directed[imports].gonum(cycles[i][0], cycles[i][1:]...))
	}
	require.Equal(t, 2, len(cycles))
	require.Equal(t, Path{A, B, A}, cycles[0])
	require.Equal(t, Path{B, C, B}, cycles[1])
}

func TestDirectedSort(t *testing.T) {

	g := Builder(Options{})
	next := EdgeKind(1)
	prev := EdgeKind(2)

	m := 1000
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
	//t.Log(forward)

	from := g.Node(NodeKey("N1"))
	to := g.Node(NodeKey("N3"))

	exists, err := PathExistsIn(g, next, from, to)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = PathExistsIn(g, EdgeKind(0), from, to)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = PathExistsIn(g, next, g.Node(NodeKey("N1")), last)
	require.NoError(t, err)
	require.False(t, exists)

}

func TestGraphQueries(t *testing.T) {

	g := Builder(Options{})
	likes := EdgeKind(1)

	g.Add(&nodeT{id: "David"})

	m := 1000
	for i := 0; i < m*2; i++ {
		this := &nodeT{id: fmt.Sprintf("LIKED-%v", i)}
		g.Add(this)
		g.Associate(g.Node(NodeKey("David")), likes, this)
	}

	for i := 0; i < m; i++ {
		that := &nodeT{id: fmt.Sprintf("LIKER-%v", i)}
		g.Add(that)
		g.Associate(that, likes, g.Node(NodeKey("David")))
	}

	require.Equal(t, m, len(NodeSlice(g.To(g.Node("David"), likes).Nodes())))
	require.Equal(t, m*2, len(NodeSlice(g.From(g.Node("David"), likes).Nodes())))
}
