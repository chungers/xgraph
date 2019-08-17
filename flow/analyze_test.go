package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

func TestAnalyze(t *testing.T) {
	deps := xg.EdgeKind(1)
	gg := testBuildGraph(deps)

	ref := GraphRef("test1")
	ordered, err := xg.DirectedSort(gg, deps)
	require.NoError(t, err)

	options := Options{
		Logger: nologging{},
	}
	g, err := analyze(ref, gg, deps, ordered, options)
	require.NoError(t, err)
	require.NotNil(t, g)

	require.NotNil(t, g.Node)
	require.Equal(t, len(ordered)+1, len(g.ordered))
	require.Equal(t, 1, len(g.output))
	require.Equal(t, 5, len(g.input))

	require.Equal(t, xg.NodeSlice{
		gg.Node(xg.NodeKey("x1")),
		gg.Node(xg.NodeKey("x2")),
		gg.Node(xg.NodeKey("x3")),
		gg.Node(xg.NodeKey("y1")),
		gg.Node(xg.NodeKey("y2")),
	}, g.inputNodes())

	require.Equal(t, xg.NodeSlice{
		gg.Node(xg.NodeKey("ratio")),
	}, g.outputNodes())

}

func TestAnalyzePairs(t *testing.T) {

	keys := xg.NodeSlice{
		&nodeT{id: "1"},
		&nodeT{id: "2"},
		&nodeT{id: "3"},
	}
	chs := []<-chan work{
		allocWorkChan(),
		allocWorkChan(),
		allocWorkChan(),
	}
	m := map[xg.Node]<-chan work{}
	for i := range keys {
		m[keys[i]] = chs[i]
	}

	k, c := pairs(m)
	require.Equal(t, len(keys), len(k))
	require.Equal(t, len(chs), len(c))

	for i := range keys {
		for j := range k {
			if keys[i] == k[j] {
				k[j] = nil
			}
		}
	}
	require.Equal(t, xg.NodeSlice{nil, nil, nil}, k)

	for i := range chs {
		for j := range c {
			if chs[i] == c[j] {
				c[j] = nil
			}
		}
	}
	require.Equal(t, []<-chan work{nil, nil, nil}, c)
}
