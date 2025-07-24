package main

import "context"

func Level1(ctx context.Context) error {

	return Level2(ctx)
}

func Level2(ctx context.Context) error {
	// comment
	return Level3(ctx)
}

func Level3(ctx context.Context) error {
	panic("line 15")
}

func main() {
	Level1(context.Background())
}
