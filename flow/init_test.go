package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitEdgeSorter(t *testing.T) {
	attr := attributes{}

	sorter := edgeSorter("undefined")
	require.NotNil(t, sorter)

	sorter = edgeSorter(attr.EdgeSorter)
	require.NotNil(t, sorter)

	require.NotNil(t, edgeSorters["edge_attr_order_or_node_key"])
}
