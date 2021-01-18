package util

import (
	"strings"

	"github.com/gofaith/goctlr/api/spec"
)

func GetAnnotationValue(annos []spec.Annotation, key, field string) (string, bool) {
	for _, anno := range annos {
		if anno.Name == key {
			value, ok := anno.Properties[field]
			return strings.TrimSpace(value), ok
		}
	}
	return "", false
}
