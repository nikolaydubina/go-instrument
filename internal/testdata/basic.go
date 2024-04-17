package example

import (
	"context"
)

func AnonymousFuncWithoutContext() func() (name string, err error) {
	return func() (name string, err error) {
		return "fluffer", nil
	}
}

func AnonymousFunc() func(ctx context.Context) (name string, err error) {
	return func(ctx context.Context) (name string, err error) {
		return "fluffer", nil
	}
}

func AnonymousFuncSkippedNoContext(ctx context.Context) func() (name string, err error) {
	return func() (name string, err error) {
		return "fluffer", nil
	}
}

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

//instrument:include Basic|Fib
//instrument:include Basic

func Basic(ctx context.Context) (err error) {
	return nil
}

func Comment(ctx context.Context) int {
	// some-comment first line
	// some-comment second line
	return 43
}

func Skip(ctx context.Context) {}

func SkipTwo(ctx context.Context) {
	//instrument:exclude SkipTwo
}

func WillNotSkipThree(ctx context.Context) { /* instrument:excluce SkipThree */ }

//instrument:exclude Skip|Something

// unmatched
//instrument:include ASDFASDFASDF

// regexp is treated as literal string
//instrument:include .*

// instrument:exclude WillNotSkipFour
func WillNotSkipFour(ctx context.Context) {}

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

func OneLineTypical(ctx context.Context, n int) (int, error) { return fib(n), nil }

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

func FunctionCallingAnonymousFunc(ctx context.Context) error {
	if err := Exec(ctx, func(ctx context.Context) error {
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func Exec(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
