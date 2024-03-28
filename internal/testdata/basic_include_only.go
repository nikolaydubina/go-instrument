package example

import (
	"context"
)

//instrument:include Bark

type Dog struct{}

func (s Dog) Bark(ctx context.Context) (name string, err error) {
	return "spot", nil
}

type Racoon struct{}

func (s Racoon) Shh(ctx context.Context, a int) (err error) {
	return nil
}
