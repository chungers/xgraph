package xgraph // import "github.com/orkestr8/xgraph"

type intNode int64

type nodeT struct {
	id     string
	custom interface{}
}

func (n *nodeT) NodeKey() NodeKey {
	return NodeKey(n.id)
}

func (n *nodeT) String() string {
	return n.id
}
