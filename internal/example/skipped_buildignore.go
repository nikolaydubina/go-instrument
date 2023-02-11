//go:build ignore
// +build ignore

package example

import (
	"context"
)

func Skip2(ctx context.Context) (name string, err error) {
	return "asdf", nil
}
