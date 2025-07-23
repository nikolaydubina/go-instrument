package main

import "context"

// Simple panic function for basic line number testing
func TestFunc(ctx context.Context) error {
	// This is line 7
	// This is line 8
	panic("test panic on line 9") // Line 9
}

func main() {
	TestFunc(context.Background())
}