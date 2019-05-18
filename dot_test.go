package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/encoding"
)

func TestEncodeDot(t *testing.T) {

	likes := EdgeKind(1)
	shares := EdgeKind(2)

	dotOptions := DotOptions{
		Name:      "V",
		Indent:    "  ",
		NodeShape: NodeShapeBox,
		Edges: map[EdgeKind]string{
			likes:  "likes",
			shares: "shares",
		},
		EdgeColors: map[EdgeKind]EdgeColor{
			likes:  EdgeColorRed,
			shares: EdgeColorBlue,
		},
	}

	A := &nodeT{id: "A"}
	B := &nodeT{id: "B"}
	C := &nodeT{id: "C"}
	D := &nodeT{id: "D"}

	g := Builder(Options{})
	g.Add(A, B, C, D)

	_, is := g.(*graph).Node(0).(encoding.Attributer)
	require.True(t, is)

	g.Associate(A, likes, B)
	g.Associate(A, likes, C)
	g.Associate(A, likes, D)

	g.Associate(B, shares, A, "ba", "xx",
		func(e Edge) string { return fmt.Sprintf("%v %v %v", e.From(), dotOptions.Edges[e.Kind()], e.To()) })
	g.Associate(B, shares, C, "bc")
	g.Associate(B, shares, D, "bd")

	g.Associate(C, shares, B)
	g.Associate(C, shares, D)
	g.Associate(C, shares, A)

	buff, err := RenderDot(g, dotOptions)
	require.NoError(t, err)
	fmt.Println(string(buff))
}
