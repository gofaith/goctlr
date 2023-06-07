package gingen

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gofaith/goctlr/api/spec"
	apiutil "github.com/gofaith/goctlr/api/util"
	"github.com/gofaith/goctlr/util"
)

const (
	handlerTemplate = `package {{.package}}

import (
	"{{.module}}/internal/types"
	"github.com/gin-gonic/gin"
)

const {{.handlerName}}Path = "{{.route}}"

func {{.handlerName}}(c *gin.Context) { {{if ne .request ""}}
	var r types.{{.request}}
	e := c.ShouldBind(&r){{end}}
{{if ne .response ""}}
	var res types.{{.response}}
{{end}}
}
`
)

func genHandler(dir string, group spec.Group, route spec.Route) error {
	handler, ok := apiutil.GetAnnotationValue(route.Annotations, "server", "handler")
	if !ok {
		return fmt.Errorf("missing handler annotation for %q", route.Path)
	}
	handler = getHandlerName(handler)

	return doGenToFile(dir, handler, group, route)
}

func doGenToFile(dir, handler string, group spec.Group, route spec.Route) error {
	folderPath := getHandlerFolderPath(group, route)
	if folderPath != handlerDir {
		handler = strings.Title(handler)
	}
	parentPkg, err := getParentPackage(dir)
	if err != nil {
		return err
	}
	filename := strings.ToLower(handler)
	if strings.HasSuffix(filename, "handler") {
		filename = filename + ".go"
	} else {
		filename = filename + "handler.go"
	}
	fp, created, err := apiutil.MaybeCreateFile(dir, folderPath, filename)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	defer fp.Close()
	t := template.Must(template.New("handlerTemplate").Parse(handlerTemplate))
	buffer := new(bytes.Buffer)
	err = t.Execute(buffer, map[string]string{
		"package":     filepath.Base(folderPath),
		"module":      parentPkg,
		"handlerName": handler,
		"request":     route.RequestType.Name,
		"response":    route.ResponseType.Name,
	})
	if err != nil {
		return nil
	}
	formatCode := formatCode(buffer.String())
	_, err = fp.WriteString(formatCode)
	return err
}

func genHandlers(dir string, api *spec.ApiSpec) error {
	for _, group := range api.Service.Groups {
		for _, route := range group.Routes {
			if err := genHandler(dir, group, route); err != nil {
				return err
			}
		}
	}

	return nil
}

func getHandlerBaseName(handler string) string {
	handlerName := util.Untitle(handler)
	if strings.HasSuffix(handlerName, "handler") {
		handlerName = strings.ReplaceAll(handlerName, "handler", "")
	} else if strings.HasSuffix(handlerName, "Handler") {
		handlerName = strings.ReplaceAll(handlerName, "Handler", "")
	}
	return handlerName
}

func getHandlerFolderPath(group spec.Group, route spec.Route) string {
	folder, ok := apiutil.GetAnnotationValue(route.Annotations, "server", folderProperty)
	if !ok {
		folder, ok = apiutil.GetAnnotationValue(group.Annotations, "server", folderProperty)
		if !ok {
			return handlerDir
		}
	}
	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")
	return path.Join(handlerDir, folder)
}

func getHandlerName(handler string) string {
	return getHandlerBaseName(handler) + "Handler"
}
