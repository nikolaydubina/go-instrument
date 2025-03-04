package example

import (
	"context"
)

type ContextHolder struct {
	ctx context.Context
}

// Should be instrumented - context in receiver only
func (c ContextHolder) MethodWithoutContextParam() (err error) {
	return nil
}

// Should be skipped - context in both receiver and params
func (c ContextHolder) MethodWithContextParam(ctx context.Context) (err error) {
	return nil
}

// Another structure to test
type MultiContextReceiver struct {
	ctx1 context.Context
	ctx2 context.Context
}

// Should not be instrumented - has multiple context in receiver
func (m MultiContextReceiver) MethodWithReceiverContext() (err error) {
	return nil
}
