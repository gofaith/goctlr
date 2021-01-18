package parser

import "github.com/gofaith/goctlr/api/spec"

type state interface {
	process(api *spec.ApiSpec) (state, error)
}
