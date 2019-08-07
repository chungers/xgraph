package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
)

type contextKeyType int

var (
	nullFlowID = flowID(nil)
)

const (
	flowIDContextKey contextKeyType = iota
	loggerContextKey

	// operatorContextKey
	// nullFlowOperator = "__null__"
)

func setLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, l)
}

func setFlowID(ctx context.Context, id flowID) context.Context {
	return context.WithValue(ctx, flowIDContextKey, id)
}

// func setFlowOperator(ctx context.Context, op string) context.Context {
// 	return context.WithValue(ctx, operatorContextKey, op)
// }

func flowIDFrom(ctx context.Context) flowID {
	v, is := ctx.Value(flowIDContextKey).(flowID)
	if is {
		return v
	}
	return nullFlowID
}

// func flowOperatorFrom(ctx context.Context) string {
// 	v, is := ctx.Value(operatorContextKey).(string)
// 	if is {
// 		return v
// 	}
// 	return nullFlowOperator
// }

func loggerFrom(ctx context.Context) Logger {
	v, is := ctx.Value(loggerContextKey).(Logger)
	if is {
		return v
	}
	return logger(0)
}
