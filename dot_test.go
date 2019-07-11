package xgraph // import "github.com/orkestr8/xgraph"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func testDataVpc(t *testing.T) Graph {

	g := Builder(Options{})

	contains := EdgeKind(1)
	depends := EdgeKind(2)

	VPC := &nodeT{id: "VPC", custom: map[string]interface{}{}}

	g.Add(VPC)

	g.Add(&nodeT{id: "sg1"}, &nodeT{id: "sg2"})

	az := []string{"az1", "az2"}

	// Technique here uses embedded types to mark the different nodes types
	// this will make it easy to look for to/from nodes of certain types
	// after the edge is known.
	type subnet_t struct {
		*nodeT
	}

	type host_t struct {
		*nodeT
	}

	type device_t struct {
		*nodeT
	}

	subnets := []*subnet_t{}
	for i := 0; i < 4; i++ {

		subnet := &subnet_t{
			&nodeT{
				id: fmt.Sprintf("subnet-%v", i),
				custom: map[string]interface{}{
					"vpc": VPC,
					"az":  az[i%len(az)],
				},
			},
		}

		subnets = append(subnets, subnet)

		g.Add(subnet)
		g.Associate(VPC, contains, subnet)
		g.Associate(subnet, depends, VPC)
	}

	hosts := []*host_t{}
	for i := range subnets {

		ipMask := fmt.Sprintf("10.0.%d.%s", i, "%d")

		for j := 0; j < 4; j++ {

			host := &host_t{&nodeT{
				id: fmt.Sprintf("host-%v-%v", i, j),
				custom: map[string]interface{}{
					"subnet": subnets[i],
					"ip":     fmt.Sprintf(ipMask, j),
				},
			}}

			g.Add(host)
			g.Associate(host, depends, subnets[i])
			g.Associate(subnets[i], contains, host)

			hosts = append(hosts, host)
		}
	}

	disks := []*device_t{}
	for i := range hosts {

		disk := &device_t{&nodeT{
			id: fmt.Sprintf("disk-%d", i),
			custom: map[string]interface{}{
				"host":  hosts[i],
				"mount": "/dev/sd1",
			},
		}}

		disks = append(disks, disk)

		g.Add(disk)
		g.Associate(hosts[i], contains, disk)

		// find the subnet of the host
		onlySubnet := func(n Node) bool {
			_, is := n.(*subnet_t)
			return is
		}

		sn := g.To(contains, hosts[i]).Nodes(onlySubnet).Slice()[0]
		g.Associate(disk, depends, sn)

	}

	// verify all disks have a dependency on the subnet
	for _, d := range disks {

		subnet, is := g.From(d, depends).Nodes().Slice()[0].(*subnet_t)
		require.True(t, is)

		onlyDisks := func(n Node) bool {
			_, is := n.(*device_t)
			return is
		}
		require.Equal(t, 4, len(g.To(depends, subnet).Nodes(onlyDisks).Slice()))
	}

	return g
}

func TestEncodeDotVpc(t *testing.T) {

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

	buff, err := EncodeDot(g, dotOptions)
	require.NoError(t, err)
	//fmt.Println(string(buff))
	t.Log(string(buff))
}

func TestDecodeDot(t *testing.T) {

	dot := `
strict digraph V {
  edge [
    color=blue
    kind = input
  ];

  // Node definitions.
  x1
  x2
  x3
  y1
  y2
  sumX [op=sum, label=sum];
  sumY [
        op=sum, label=sum,
        shape=doublecircle,
        service=remote_inference
       ];
  ratio [op=ratio];

  // Edge definitions.
  sumX -> ratio;
  sumY -> ratio;
  x1 -> sumX;
  x2 -> sumX;
  x3 -> sumX;
  x3 -> sumY [context=0, signal=found];
  y1 -> sumY [context=1];
  y2 -> sumY [context=2];
}
`
	t.Log(dot)

	g := Builder(Options{})

	kind := EdgeKind(0)
	err := DecodeDot([]byte(dot), g, kind)
	require.NoError(t, err)

	// Now Encode
	dotOptions := DotOptions{
		Name:      "V",
		Indent:    "  ",
		NodeShape: NodeShapeBox,
		Edges: map[EdgeKind]string{
			kind: "input",
		},
		EdgeColors: map[EdgeKind]EdgeColor{
			kind: EdgeColorRed,
		},
		EdgeLabelers: map[Edge]EdgeLabeler{},
		NodeLabelers: map[Node]NodeLabeler{},
	}
	view, err := EncodeDot(g, dotOptions)
	require.NoError(t, err)
	t.Log(string(view))
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

	g.Associate(A, likes, B)
	g.Associate(A, likes, C)
	g.Associate(A, likes, D)

	g.Associate(B, shares, A,
		Attribute{Key: "key1", Value: "ba"}, Attribute{Key: "key2", Value: "xx"})
	g.Associate(B, shares, C,
		Attribute{Key: "key1", Value: "bc"})
	g.Associate(B, shares, D,
		Attribute{Key: "key1", Value: "bd"})

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

	buff, err := EncodeDot(g, dotOptions)
	require.NoError(t, err)
	//fmt.Println(string(buff))
	t.Log(string(buff))

	dotOptions.EdgeLabelers = nil
	_, err = EncodeDot(g, dotOptions)
	require.NoError(t, err)
}

func TestDotEdgeLabels(t *testing.T) {

	ed := &dotEdge{
		edge: &edge{},
	}
	require.Equal(t, "", ed.label())

	ed = &dotEdge{
		edge: &edge{
			attributes: []Attribute{
				{Key: "foo", Value: "bar"},
			},
		},
	}
	require.Equal(t, "", ed.label())

	ed = &dotEdge{
		edge: &edge{
			attributes: []Attribute{
				{Key: "label", Value: "bar"},
			},
		},
	}
	require.Equal(t, "bar", ed.label())

	label := "my label"
	ed = &dotEdge{
		edge: &edge{
			attributes: []Attribute{
				{
					Key: "whatever",
					Value: func(edge Edge) string {
						return label
					},
				},
			},
		},
	}
	require.Equal(t, label, ed.label())

	label2 := "my label2"
	ed = &dotEdge{
		edge: &edge{
			attributes: []Attribute{
				{
					Key: "foo",
					Value: func(edge Edge) string {
						return label
					},
				},
				{
					Key: "bar",
					Value: func(edge Edge) string {
						return label2
					},
				},
			},
		},
	}
	require.Equal(t, label+","+label2, ed.label())
}
