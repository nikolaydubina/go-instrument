//go:build ignore

package example

import (
	"context"
)

func Skip1(ctx context.Context) (name string, err error) {
	return "asdf", nil
}
