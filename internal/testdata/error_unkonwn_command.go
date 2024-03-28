package example

import (
	"context"
)

//instrument:asdf
func Skip5(ctx context.Context) (name string, err error) {
	return "asdf", nil
}
