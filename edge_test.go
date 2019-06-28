package xgraph // import "github.com/orkestr8/xgraph"

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDotEdgeLabel(t *testing.T) {

	var ed *dotEdge

	ed = &dotEdge{
		edge: &edge{
			context: []interface{}{},
		},
	}

	require.Equal(t, "", ed.label())

	ed = &dotEdge{
		edge: &edge{
			context: []interface{}{
				"foo", "bar",
			},
		},
	}
	require.Equal(t, "", ed.label())

	label := "my label"
	ed = &dotEdge{
		edge: &edge{
			context: []interface{}{
				func(edge Edge) string {
					return label
				},
			},
		},
	}
	require.Equal(t, label, ed.label())

	label2 := "my label2"
	ed = &dotEdge{
		edge: &edge{
			context: []interface{}{
				func(edge Edge) string {
					return label
				},
				func(edge Edge) string {
					return label2
				},
			},
		},
	}
	require.Equal(t, strings.Join([]string{label, label2}, ","), ed.label())
}
