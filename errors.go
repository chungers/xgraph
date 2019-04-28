package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
)

type ErrDuplicateKey struct {
	Node
}

func (e ErrDuplicateKey) Error() string {
	return fmt.Sprintf("Duplicate key:%s", e.Node.NodeKey())
}

type ErrNoSuchNode struct {
	Node
	context string
}

func (e ErrNoSuchNode) Error() string {
	return fmt.Sprintf("Missing %s node:%s", e.context, e.Node.NodeKey())
}

type ErrNotSupported struct {
	Graph
}

func (e ErrNotSupported) Error() string {
	return fmt.Sprintf("Not supported: %v", e.Graph)
}
