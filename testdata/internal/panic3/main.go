package main

import "context"

func Level1(ctx context.Context) func() error {
	return func() error {
		return Level2(ctx, 0)
	}
}

func Level2(ctx context.Context, i int) error {
	if i >= 5 {
		return Level3(ctx)
	}
	return Level2(ctx, i+1)
}

func Level3(ctx context.Context) error {
	panic("line 19")
}

func FuncWithBody(ctx context.Context) error {
	i := 1

	for j := i; j < 100; j++ {
		if j%13 == 2 {
			return Level1(ctx)()
		}
	}

	return nil
}

func main() {
	FuncWithBody(context.Background())
}
