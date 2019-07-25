package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAttributes(t *testing.T) {

	a := &attributes{}
	require.NoError(t, a.unmarshal(nil))

	timeout := 20 * time.Second
	require.NoError(t, a.unmarshal(map[string]interface{}{
		"timeout": Duration(timeout),
	}))
	require.Equal(t, Duration(timeout), a.Timeout)
}
