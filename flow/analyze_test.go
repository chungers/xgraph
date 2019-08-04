package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"testing"

	xg "github.com/orkestr8/xgraph"
	"github.com/stretchr/testify/require"
)

func TestAnalyze1(t *testing.T) {
	deps := xg.EdgeKind(1)
	gg := testBuildGraph(deps)

	ref := GraphRef("test1")
	ordered, err := xg.DirectedSort(gg, deps)
	require.NoError(t, err)

	options := Options{
		Logger: logger(1),
	}
	g, err := analyze(ref, gg, deps, ordered, options)
	require.NoError(t, err)
	require.NotNil(t, g)
}

func TestAnalyzePairs(t *testing.T) {

	keys := xg.NodeSlice{
		&nodeT{id: "1"},
		&nodeT{id: "2"},
		&nodeT{id: "3"},
	}
	chs := []<-chan work{
		make(chan work),
		make(chan work),
		make(chan work),
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
