package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"context"
)

type contextKeyType int

const (
	flowIDContextKey contextKeyType = iota
	loggerContextKey
	gatherChanContextKey
	awaitableContextKey
)

func setLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, l)
}

func setFlowID(ctx context.Context, id flowID) context.Context {
	return context.WithValue(ctx, flowIDContextKey, id)
}

func setGatherChan(ctx context.Context, ch chan gather) context.Context {
	return context.WithValue(ctx, gatherChanContextKey, ch)
}

func setAwaitable(ctx context.Context, aw Awaitable) context.Context {
	return context.WithValue(ctx, awaitableContextKey, aw)
}

func flowIDFrom(ctx context.Context) flowID {
	v, is := ctx.Value(flowIDContextKey).(flowID)
	if is {
		return v
	}
	return nil
}

func gatherChanFrom(ctx context.Context) chan gather {
	v, is := ctx.Value(gatherChanContextKey).(chan gather)
	if is {
		return v
	}
	return nil
}

func awaitableFrom(ctx context.Context) Awaitable {
	v, is := ctx.Value(awaitableContextKey).(Awaitable)
	if is {
		return v
	}
	return nil
}

func loggerFrom(ctx context.Context) Logger {
	v, is := ctx.Value(loggerContextKey).(Logger)
	if is {
		return v
	}
	return logger(0)
}
