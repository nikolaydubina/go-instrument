package example

import "github.com/nikolaydubina/go-instrument/internal/example/custom"

func CustomAnonymousFunc() func(c *custom.Context) (name string, err error) {
	return func(c *custom.Context) (name string, err error) {
		return "fluffer", nil
	}
}

type CustomCat struct{}

func (s CustomCat) Name(ctx *custom.Context) (name string, err error) {
	return "fluffer", nil
}

type exampleContext struct {
	context *custom.Context
}

func CustomApplicationContext(ctx *exampleContext, appName string) (name string, err error) {
	return "fluffer", nil
}
