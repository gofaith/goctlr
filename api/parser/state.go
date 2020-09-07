package parser

import "github.com/gofaith/goctl/api/spec"

type state interface {
	process(api *spec.ApiSpec) (state, error)
}
