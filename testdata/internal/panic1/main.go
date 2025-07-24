package main

import "context"

func TestFunc(ctx context.Context) error {
	panic("line 6")
}

func main() {
	TestFunc(context.Background())
}
