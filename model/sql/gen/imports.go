package gen

import (
	"github.com/gofaith/goctlr/model/sql/template"
	"github.com/gofaith/goctlr/util"
)

func genImports(withCache, timeImport bool) (string, error) {
	if withCache {
		buffer, err := util.With("import").Parse(template.Imports).Execute(map[string]interface{}{
			"time": timeImport,
		})
		if err != nil {
			return "", err
		}
		return buffer.String(), nil
	} else {
		buffer, err := util.With("import").Parse(template.ImportsNoCache).Execute(map[string]interface{}{
			"time": timeImport,
		})
		if err != nil {
			return "", err
		}
		return buffer.String(), nil
	}
}
