package example

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Apple struct{}

func (s *Apple) MethodWithPointerReciver(ctx context.Context, a int) (err error) {
	ctx, span := otel.Trace("my-service").Start(ctx, "Apple.MethodWithPointerReciver")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, "error")
			span.RecordError(err)
		}
	}()
	return nil
}

func (s Apple) MethodWithValueReciver(ctx context.Context, a int) (err error) {
	ctx, span := otel.Trace("my-service").Start(ctx, "Apple.MethodWithValueReciver")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, "error")
			span.RecordError(err)
		}
	}()
	return nil
}

func (*Apple) MethodWithPointerReciverUnnamed(ctx context.Context, a int) (err error) {
	ctx, span := otel.Trace("my-service").Start(ctx, "Apple.MethodWithPointerReciverUnnamed")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, "error")
			span.RecordError(err)
		}
	}()
	return nil
}

func (Apple) MethodWithValueReciverUnnamed(ctx context.Context, a int) (err error) {
	ctx, span := otel.Trace("my-service").Start(ctx, "Apple.MethodWithValueReciverUnnamed")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, "error")
			span.RecordError(err)
		}
	}()
	return nil
}

func Fib(ctx context.Context, n int) int {
	ctx, span := otel.Trace("my-service").Start(ctx, "Fib")
	defer span.End()
	if n == 0 || n == 1 {
		return 1
	}
	return Fib(ctx, n-1) + Fib(ctx, n-2)
}

func Basic(ctx context.Context) (err error) {
	ctx, span := otel.Trace("my-service").Start(ctx, "Basic")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, "error")
			span.RecordError(err)
		}
	}()
	return nil
}

func Comment(ctx context.Context) int {
	ctx, span := otel.Trace("my-service").Start(ctx, "Comment")
	defer span.End()

	return 43
}

func CommentMultiline() error {

	return nil
}

func fib(n int) int {
	if n == 0 || n == 1 {
		return 1
	}
	return fib(n-1) + fib(n-2)
}

func OneLine(n int) int	{ return fib(n) }

func OneLineTypical(ctx context.Context) error {
	ctx, span := otel.Trace("my-service").Start(ctx, "OneLineTypical")
	defer span.End()
	return nil
}

func OneLineWithComment() int	{ return 42 }

func CustomName(b int, specialCtx context.Context) (specialErr error) {
	return nil
}

func MultipleContextMultipleError(a context.Context, b context.Context) (erra error, errorb error) {
	return nil, nil
}

func MultipleContextMultipleErrorCollapsed(a, b context.Context) (erra, errob error) {
	return nil, nil
}

func MultipleErrorNotNamed(ctx context.Context) (error, error) {
	ctx, span := otel.Trace("my-service").Start(ctx, "MultipleErrorNotNamed")
	defer span.End()
	return nil, nil
}

func Closure(ctx context.Context) (int, error) {
	ctx, span := otel.Trace("my-service").Start(ctx, "Closure")
	defer span.End()
	a := func(x int) (int, error) { return x + 1, nil }
	return a(5)
}
