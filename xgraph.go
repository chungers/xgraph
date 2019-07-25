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

// // NodeSlice reads all the nodes from the channel until closed and then returns the entire slice of nodes collected.
// func NodeSlice(nodes Nodes) []Node {
// 	all := []Node{}
// 	for n := range nodes {
// 		all = append(all, n)
// 	}
// 	return all
// }

// // EdgeSlice reads all the edges from the channel until closed and then returns the entire slice of edges collected.
// func EdgeSlice(edges Edges) []Edge {
// 	all := []Edge{}
// 	for n := range edges {
// 		all = append(all, n)
// 	}
// 	return all
// }
