package main

import "context"

// Nested function calls for testing call stack line numbers
func Level1(ctx context.Context) error {
	// This is line 7
	return Level2(ctx) // Line 8
}

func Level2(ctx context.Context) error {
	// This is line 12
	return Level3(ctx) // Line 13
}

func Level3(ctx context.Context) error {
	// This is line 17
	// This is line 18
	panic("nested panic on line 19") // Line 19
}

func main() {
	Level1(context.Background())
}