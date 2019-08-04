package flow // import "github.com/orkestr8/xgraph/flow"

import (
	xg "github.com/orkestr8/xgraph"
)

var (
	edgeSorters map[string]func(xg.Edge, xg.Edge) bool = map[string]func(xg.Edge, xg.Edge) bool{
		"edge_attr_order_or_node_key": OrderEdgesByEdgeAttributeOrderOrNodeKey,
	}
)

func edgeSorter(key string) func(xg.Edge, xg.Edge) bool {
	if s, has := edgeSorters[key]; has {
		return s
	}
	return OrderEdgesByEdgeAttributeOrderOrNodeKey
}
