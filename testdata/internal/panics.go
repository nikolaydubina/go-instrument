package main

import "context"

// Simple panic function for basic line number testing
func TestFunc(ctx context.Context) error {
	// This is line 7
	// This is line 8
	panic("test panic on line 9") // Line 9
}

// Nested function calls for testing call stack line numbers
func Level1(ctx context.Context) error {
	// This is line 14
	return Level2(ctx) // Line 15
}

func Level2(ctx context.Context) error {
	// This is line 19
	return Level3(ctx) // Line 20
}

func Level3(ctx context.Context) error {
	// This is line 24
	// This is line 25
	panic("nested panic on line 26") // Line 26
}

// Function with complex body containing loops and conditionals
func FuncWithBody(ctx context.Context) error {
	i := 1

	for j := i; j < 100; j++ {
		if j%13 == 2 {
			return Level3(ctx) // This should report correct line number
		}
	}

	return nil
}
