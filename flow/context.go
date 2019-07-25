package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
)

type contextKeyType int

const (
	flowIDContextKey contextKeyType = iota
	operatorContextKey

	nullFlowID       = flowID(-1)
	nullFlowOperator = "__null__"
)

func setFlowID(ctx context.Context, id flowID) context.Context {
	return context.WithValue(ctx, flowIDContextKey, id)
}

func setFlowOperator(ctx context.Context, op string) context.Context {
	return context.WithValue(ctx, operatorContextKey, op)
}
func flowIDFrom(ctx context.Context) flowID {
	v, is := ctx.Value(flowIDContextKey).(flowID)
	if is {
		return v
	}
	return nullFlowID
}

func flowOperatorFrom(ctx context.Context) string {
	v, is := ctx.Value(operatorContextKey).(string)
	if is {
		return v
	}
	return nullFlowOperator
}
