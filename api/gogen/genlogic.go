package gogen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/StevenZack/tools/strToolkit"
	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/api/util"
	apiutil "github.com/gofaith/goctlr/api/util"
	ctlutil "github.com/gofaith/goctlr/util"
	"github.com/gofaith/goctlr/vars"
)

const (
	logicTemplate = `package logic

import (
	{{.imports}}
)

type {{.logic}} struct {
	logx.Logger
	ctx    context.Context
	s *svc.ServiceContext
}

func New{{.logic}}(ctx context.Context, svcCtx *svc.ServiceContext) {{.logic}} {
	return {{.logic}}{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		s:      svcCtx,
	}
}

// {{.summary}} {{if ne .desc ""}}
// {{.desc}}{{end}}
func (l *{{.logic}}) {{.function}}({{.request}}) {{.responseType}} {
	// TODO: add your logic here and delete this line

	{{.returnString}}
}
`
	delimiter1 = `
		s:      svcCtx,
	}
}
`
	delimiter2 = `
func (l *`
)

func genLogic(dir, proto string, api *spec.ApiSpec) error {
	for _, g := range api.Service.Groups {
		for _, r := range g.Routes {
			err := genLogicByRoute(dir, g, r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func genLogicByRoute(dir string, group spec.Group, route spec.Route) error {
	handler, ok := util.GetAnnotationValue(route.Annotations, "server", "handler")
	if !ok {
		return fmt.Errorf("missing handler annotation for %q", route.Path)
	}
	typ, _ := apiutil.GetAnnotationValue(route.Annotations, "server", "type")
	summary, _ := apiutil.GetAnnotationValue(route.Annotations, "doc", "summary")
	desc, _ := apiutil.GetAnnotationValue(route.Annotations, "doc", "desc")
	handler = strings.TrimSuffix(handler, "handler")
	handler = strings.TrimSuffix(handler, "Handler")
	filename := strings.ToLower(handler)
	goFile := filename + "logic.go"
	logic := strings.Title(handler) + "Logic"
	fp, created, err := util.MaybeCreateFile(dir, getLogicFolderPath(group, route), goFile)
	if err != nil {
		return err
	}

	if !created {
		//update doc
		path := filepath.Join(dir, getLogicFolderPath(group, route), goFile)
		b, e := ioutil.ReadFile(path)
		if e != nil {
			return e
		}
		content := string(b)
		buf := new(bytes.Buffer)
		buf.WriteString(strToolkit.SubBefore(content, delimiter1, content))
		buf.WriteString(delimiter1)

		//code
		rcontent := strToolkit.SubAfter(content, delimiter1, content)
		e = strToolkit.RangeLines(strToolkit.SubBefore(rcontent, delimiter2, rcontent), func(line string) bool {
			if line == "" || strings.HasPrefix(line, "//") {
				return false
			}
			buf.WriteString("\n" + line)
			return false
		})
		if e != nil {
			return e
		}

		buf.WriteString("\n// " + summary)
		if desc != "" {
			buf.WriteString("\n// " + desc)
		}
		buf.WriteString(delimiter2)
		buf.WriteString(strToolkit.SubAfter(content, delimiter2, content))

		e = ioutil.WriteFile(path, buf.Bytes(), 0644)
		if e != nil {
			return e
		}
		return nil
	}
	defer fp.Close()

	parentPkg, err := getParentPackage(dir)
	if err != nil {
		return err
	}

	imports := genLogicImports(route, parentPkg, typ)
	var responseString string
	var returnString string
	var requestString string
	switch typ {
	case SERVER_TYPE_HTML:
		if len(route.RequestType.Name) > 0 {
			requestString = "w http.ResponseWriter, r *http.Request, req " + "types." + strings.Title(route.RequestType.Name)
		} else {
			requestString = "w http.ResponseWriter, r *http.Request"
		}

		responseString = "error"
		returnString = "return nil"
	default:
		if len(route.RequestType.Name) > 0 {
			requestString = "req " + "types." + strings.Title(route.RequestType.Name)
		}

		if len(route.ResponseType.Name) > 0 {
			resp := strings.Title(route.ResponseType.Name)
			responseString = "(*types." + resp + ", error)"
			returnString = fmt.Sprintf("return &types.%s{}, nil", resp)
		} else {
			responseString = "error"
			returnString = "return nil"
		}
	}

	t := template.Must(template.New("logicTemplate").Parse(logicTemplate))
	buffer := new(bytes.Buffer)
	err = t.Execute(fp, map[string]string{
		"imports":      imports,
		"logic":        logic,
		"summary":      summary,
		"desc":         desc,
		"function":     strings.Title(strings.TrimSuffix(handler, "Handler")),
		"responseType": responseString,
		"returnString": returnString,
		"request":      requestString,
	})
	if err != nil {
		return err
	}

	formatCode := formatCode(buffer.String())
	_, err = fp.WriteString(formatCode)
	return err
}

func getLogicFolderPath(group spec.Group, route spec.Route) string {
	folder, ok := util.GetAnnotationValue(route.Annotations, "server", folderProperty)
	if !ok {
		folder, ok = util.GetAnnotationValue(group.Annotations, "server", folderProperty)
		if !ok {
			return logicDir
		}
	}
	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")
	return path.Join(logicDir, folder)
}

func genLogicImports(route spec.Route, parentPkg, typ string) string {
	var imports []string
	imports = append(imports, `"context"`)
	switch typ {
	case SERVER_TYPE_HTML:
		imports = append(imports, `"net/http"`)
		if len(route.RequestType.Name) > 0 {
			imports = append(imports, fmt.Sprintf("\"%s\"", ctlutil.JoinPackages(parentPkg, typesDir)))
		}
	default:
		if len(route.ResponseType.Name) > 0 || len(route.RequestType.Name) > 0 {
			imports = append(imports, fmt.Sprintf("\"%s\"", ctlutil.JoinPackages(parentPkg, typesDir)))
		}
	}
	imports = append(imports, fmt.Sprintf("\"%s\"", ctlutil.JoinPackages(parentPkg, contextDir)))

	imports = append(imports, fmt.Sprintf("\"%s/go-zero/core/logx\"", vars.ProjectOpenSourceUrl))
	return strings.Join(imports, "\n\t")
}
