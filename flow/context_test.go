package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextFlowID(t *testing.T) {

	require.Nil(t, flowIDFrom(context.Background()))

	ctx := context.Background()
	id := flowID(10000)
	ctx = setFlowID(ctx, id)
	require.Equal(t, id, flowIDFrom(ctx))

}
