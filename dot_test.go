package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/encoding"
)

func testDataVpc(t *testing.T) Graph {

	g := Builder(Options{})

	contains := EdgeKind(1)
	depends := EdgeKind(2)

	VPC := &nodeT{id: "VPC", custom: map[string]interface{}{}}

	g.Add(VPC)

	az := []string{"az1", "az2"}

	subnets := []*nodeT{}
	for i := 0; i < 4; i++ {

		subnet := &nodeT{
			id: fmt.Sprintf("subnet-%v", i),
			custom: map[string]interface{}{
				"vpc": VPC,
				"az":  az[i%len(az)],
			},
		}

		subnets = append(subnets, subnet)

		g.Add(subnet)
		g.Associate(VPC, contains, subnet)
		g.Associate(subnet, depends, VPC)
	}

	hosts := []*nodeT{}
	for i := range subnets {

		ipMask := fmt.Sprintf("10.0.%d.%s", i, "%d")

		for j := 0; j < 4; j++ {

			host := &nodeT{
				id: fmt.Sprintf("host-%v-%v", i, j),
				custom: map[string]interface{}{
					"subnet": subnets[i],
					"ip":     fmt.Sprintf(ipMask, j),
				},
			}

			g.Add(host)
			g.Associate(host, depends, subnets[i])
			g.Associate(subnets[i], contains, host)

			hosts = append(hosts, host)
		}
	}

	for i := range hosts {

		gpu := &nodeT{
			id: fmt.Sprintf("gpu-%d", i),
			custom: map[string]interface{}{
				"host": hosts[i],
				"path": "/dev/gpu",
			},
		}

		g.Add(gpu)
		g.Associate(hosts[i], contains, gpu)

		disk := &nodeT{
			id: fmt.Sprintf("disk-%d", i),
			custom: map[string]interface{}{
				"host":  hosts[i],
				"mount": "/dev/sd1",
			},
		}

		g.Add(disk)
		g.Associate(hosts[i], contains, disk)
	}

	return g
}

func TestEncoderDotVpc(t *testing.T) {

	contains := EdgeKind(1)
	depends := EdgeKind(2)

	g := testDataVpc(t)

	dotOptions := DotOptions{
		Name:      "VPC",
		Indent:    "  ",
		NodeShape: NodeShapeRecord,
		Edges: map[EdgeKind]string{
			contains: "contains",
			depends:  "depends",
		},
		EdgeColors: map[EdgeKind]EdgeColor{
			depends:  EdgeColorRed,
			contains: EdgeColorBlue,
		},
		EdgeLabelers: map[Edge]EdgeLabeler{},
		NodeLabelers: map[Node]NodeLabeler{},
	}

	buff, err := RenderDot(g, dotOptions)
	require.NoError(t, err)
	fmt.Println(string(buff))

}

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
		EdgeLabelers: map[Edge]EdgeLabeler{},
		NodeLabelers: map[Node]NodeLabeler{},
	}

	A := &nodeT{id: "A", custom: "Operator1"}
	B := &nodeT{id: "B", custom: "Operator2"}
	C := &nodeT{id: "C"}
	D := &nodeT{id: "D"}

	g := Builder(Options{})
	g.Add(A, B, C, D)

	_, is := g.(*graph).Node(0).(encoding.Attributer)
	require.True(t, is)

	g.Associate(A, likes, B)
	g.Associate(A, likes, C)
	g.Associate(A, likes, D)

	g.Associate(B, shares, A, "ba", "xx")
	g.Associate(B, shares, C, "bc")
	g.Associate(B, shares, D, "bd")

	g.Associate(C, shares, B)
	g.Associate(C, shares, D)
	g.Associate(C, shares, A)

	// Default labeler
	dotOptions.NodeLabelers[nil] = func(n Node) string {
		return n.(*nodeT).id
	}

	dotOptions.NodeLabelers[g.Node(A.NodeKey())] = func(n Node) string {
		return fmt.Sprintf("%v_%v", n.(*nodeT).custom, n.(*nodeT).id)
	}

	dotOptions.NodeLabelers[B] = func(n Node) string {
		return fmt.Sprintf("{%v}-{%v}", n.(*nodeT).id, n.(*nodeT).custom)
	}

	dotOptions.EdgeLabelers[g.Edge(B, shares, A)] = func(e Edge) string {
		return fmt.Sprintf("%v %v %v", e.From(), dotOptions.Edges[e.Kind()], e.To())
	}

	buff, err := RenderDot(g, dotOptions)
	require.NoError(t, err)
	fmt.Println(string(buff))

	dotOptions.EdgeLabelers = nil
	_, err = RenderDot(g, dotOptions)
	require.NoError(t, err)
}
