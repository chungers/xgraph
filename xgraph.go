package xgraph // import "github.com/orkestr8/xgraph"

func Builder(options Options) GraphBuilder {
	return newGraph(options)
}

// Reverse reverses the slice in place and returns the slice for convenience
func Reverse(n []Node) (out []Node) {
	out = n
	for left, right := 0, len(n)-1; left < right; left, right = left+1, right-1 {
		n[left], n[right] = n[right], n[left]
	}
	return
}

func NodeSlice(nodes Nodes) []Node {
	all := []Node{}
	for {
		if n, ok := <-nodes; !ok {
			break
		} else {
			all = append(all, n)
		}
	}
	return all
}
