package example

import (
	"context"
)

type Cat struct{}

func (s Cat) Name(ctx context.Context) (name string, err error) {
	return "fluffer", nil
}

type Apple struct{}

func (s *Apple) MethodWithPointerReciver(ctx context.Context, a int) (err error) {
	return nil
}

func (s Apple) MethodWithValueReciver(ctx context.Context, a int) (err error) {
	return nil
}

func (*Apple) MethodWithPointerReciverUnnamed(ctx context.Context, a int) (err error) {
	return nil
}

func (Apple) MethodWithValueReciverUnnamed(ctx context.Context, a int) (err error) {
	return nil
}

func Fib(ctx context.Context, n int) int {
	if n == 0 || n == 1 {
		return 1
	}
	return Fib(ctx, n-1) + Fib(ctx, n-2)
}

func Basic(ctx context.Context) (err error) {
	return nil
}

func Comment(ctx context.Context) int {
	//some-comment first line
	//some-comment second line
	return 43
}

func CommentMultiline() error {
	/*
		a
		b
		c
		d
	*/
	return nil
}

func fib(n int) int {
	if n == 0 || n == 1 {
		return 1
	}
	return fib(n-1) + fib(n-2)
}

func OneLine(n int) int { return fib(n) }

func OneLineTypical(ctx context.Context) error { return nil }

func OneLineWithComment() int { /* comment 1 */ return 42 /* comment 2 */ }

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
	return nil, nil
}

func Closure(ctx context.Context) (int, error) {
	a := func(x int) (int, error) { return x + 1, nil }
	return a(5)
}
