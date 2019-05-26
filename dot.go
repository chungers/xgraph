package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
)

type dotSubgraph struct {
	*directed
	DotOptions
	dot.Subgrapher
}

func (ds *dotSubgraph) DOTID() string {
	dotid := fmt.Sprintf("%v", ds.kind)
	if ds.DotOptions.Edges != nil {
		if v, has := ds.DotOptions.Edges[ds.kind]; has {
			dotid = v
		}
		return dotid
	}
	return dotid
}

func (dg *dotSubgraph) edgeColor() string {
	c := "black"
	if dg.DotOptions.EdgeColors != nil {
		if v, has := dg.DotOptions.EdgeColors[dg.kind]; has {
			c = string(v)
		}
	}
	return c
}

func (dg *dotSubgraph) edgeLabel() string {
	return dg.DOTID()
}

func (dg *dotSubgraph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	graph = attributes{}
	node = attributes{}
	edge = attributes{"color": dg.edgeColor(), "label": dg.edgeLabel()}
	return
}

func (ds *dotSubgraph) Subgraph() gonum.Graph {
	return ds.directed
}

type dotGraph struct {
	DotOptions
	gonum.Directed

	xg *graph
}

func (dg *dotGraph) DOTID() string {
	id := dg.Name
	if id == "" {
		return "G"
	}
	return id
}

type attributes map[string]string

func (a attributes) Attributes() []encoding.Attribute {
	out := []encoding.Attribute{}
	for k, v := range a {
		out = append(out, encoding.Attribute{Key: k, Value: v})
	}
	return out
}

func (dg *dotGraph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	graph = attributes{}
	node = attributes{"shape": string(dg.DotOptions.NodeShape)}
	edge = attributes{}
	return
}

func (dg *dotGraph) Structure() []dot.Graph {
	subs := []dot.Graph{}

	for k := range dg.xg.directed {
		sg := &dotSubgraph{
			directed:   dg.xg.directed[k],
			DotOptions: dg.DotOptions,
		}
		subs = append(subs, sg)
	}
	return subs
}

func RenderDot(g Graph, options DotOptions) ([]byte, error) {

	xg, is := g.(*graph)
	if !is {
		return nil, ErrNotSupported{g}
	}

	// Set any EdgeLabelers to customize labels for Edges
	count := 0
	for edg, labeler := range options.EdgeLabelers {
		if ev, is := edg.(*edgeView); is {
			ev.labeler = labeler
			count++
		}
	}
	if count > 0 {
		defer func() {
			// reset the labeler so we don't interfere with future renders
			for edg := range options.EdgeLabelers {
				if ev, is := edg.(*edgeView); is {
					ev.labeler = nil
				}
			}
		}()
	}

	// Set any NodeLabelers to customize labels for Nodes
	count = 0

	// Check if a global labeler is set
	if labeler, has := options.NodeLabelers[nil]; has {
		count = xg.setLabelers(labeler)
	}
	for n, labeler := range options.NodeLabelers {
		if n != nil {
			if v := g.Node(n.NodeKey()); v != nil {
				if xn, is := v.(*node); is {
					xn.labeler = labeler
					count++
				}
			}
		}
	}
	if count > 0 {
		defer func() {
			xg.setLabelers(nil)
		}()
	}

	dg := &dotGraph{
		DotOptions: options,
		Directed:   simple.NewDirectedGraph(), //xg.DirectedBuilder,
		xg:         xg,
	}

	return dot.Marshal(dg, options.Name, options.Prefix, options.Indent)
}
