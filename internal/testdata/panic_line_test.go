package example

import (
	"context"
	"go.opentelemetry.io/otel"
)

func PanicTest(ctx context.Context) error {
	ctx, span := otel.Tracer("app").Start(ctx, "PanicTest")
	defer span.End()
//line internal/testdata/panic_line_test.go:9

	// Line 8: This comment should still be on line 8
	panic("panic on original line 9") // This should be line 9 in stack trace
}

func PanicTestWithCode(ctx context.Context) error {
	ctx, span := otel.Tracer("app").Start(ctx, "PanicTestWithCode")
	defer span.End()
//line internal/testdata/panic_line_test.go:14

	// Line 13: Some setup code
	x := 42
	y := x * 2
	// Line 16: The panic should be on this line
	panic("panic on original line 17") // This should be line 17 in stack trace
}
