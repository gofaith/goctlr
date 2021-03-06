package gogen

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gofaith/goctlr/api/spec"
	apiutil "github.com/gofaith/goctlr/api/util"
	"github.com/gofaith/goctlr/util"
	"github.com/gofaith/goctlr/vars"
	"github.com/iancoleman/strcase"
)

const (
	handlerTemplate = `package handler

import (
	"net/http"

	{{.importPackages}}
)

func {{.handlerName}}(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		{{.handlerBody}}
	}
}
`
	handlerBodyTemplate = `{{.parseRequest}}
		{{.processBody}}
`
	parseRequestTemplate = `var req {{.requestType}}
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}
`
	hasRespTemplate = `
		l := logic.{{.logic}}(r.Context(), ctx)
		{{.logicResponse}} l.{{.callee}}({{.req}})
		if err != nil {
			httpx.Error(w, err)
		} else {
			{{.respWriter}}
		}
	`
	hasRespTemplate_HtmlMode = `
		l := logic.{{.logic}}(r.Context(), ctx)
		err := l.{{.callee}}({{.req}})
		if err != nil {
			httpx.Error(w, err)
		}
	`
	protoTemplate = `package handler

import (
	"io"
	"net/http"

	"github.com/gofaith/rest/httpx"
	logic "{{.pkg}}/internal/logic/user"
	"{{.pkg}}/internal/pb"
	"{{.pkg}}/internal/svc"{{if and (ne .route.RequestType.Name "") (ne .route.ResponseType.Name "")}}
	"google.golang.org/protobuf/proto"{{end}}
)

func {{.name}}Handler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.New{{.name}}Logic(r.Context(), ctx){{if ne .route.RequestType.Name ""}}
		b, e := io.ReadAll(r.Body)
		if e != nil {
			httpx.Error(w, e)
			return
		}
		req := pb.User{{.name}}Request{}
		e = proto.Unmarshal(b, &req)
		if e != nil {
			httpx.Error(w, e)
			return
		}{{if ne .route.ResponseType.Name ""}}
		res, e := l.{{.name}}(&req)
		if e != nil {
			httpx.Error(w, e)
			return
		}
		b2, e := proto.Marshal(res)
		if e != nil {
			httpx.Error(w, e)
			return
		}
		w.Write(b){{else}}
		e := l.{{.name}}(&req)
		if e != nil {
			httpx.Error(w, e)
			return
		}
		httpx.Ok(w){{end}}{{else}}{{if ne .route.ResponseType.Name ""}}
		res, e := l.{{.name}}()
		if e != nil {
			httpx.Error(w, e)
			return
		}
		b2, e := proto.Marshal(res)
		if e != nil {
			httpx.Error(w, e)
			return
		}
		w.Write(b){{else}}
		e := l.{{.name}}()
		if e != nil {
			httpx.Error(w, e)
			return
		}
		httpx.Ok(w){{end}}{{end}}
	}
}
`
)

func genHandlerProto(dir string, group spec.Group, route spec.Route) error {
	handler, ok := apiutil.GetAnnotationValue(route.Annotations, "server", "handler")
	if !ok {
		return fmt.Errorf("missing handler annotation for %q", route.Path)
	}
	handler = getHandlerName(handler)
	pkg, e := getParentPackage(dir)
	if e != nil {
		log.Println(e)
		return e
	}

	t, e := template.New("protoTemplate").Parse(protoTemplate)
	if e != nil {
		log.Println(e)
		return e
	}

	base := filepath.Join(dir, getHandlerFolderPath(group, route))
	os.MkdirAll(base, 0755)
	path := filepath.Join(base, strings.ToLower(handler)+".go")
	fo, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		log.Println(e)
		return e
	}
	defer fo.Close()

	e = t.Execute(fo, map[string]interface{}{
		"pkg":   pkg,
		"route": route,
		"name":  strcase.ToCamel(strings.TrimSuffix(handler, "Handler")),
	})
	if e != nil {
		log.Println(e)
		return e
	}
	return nil
}

func genHandler(dir string, group spec.Group, route spec.Route) error {
	handler, ok := apiutil.GetAnnotationValue(route.Annotations, "server", "handler")
	if !ok {
		return fmt.Errorf("missing handler annotation for %q", route.Path)
	}
	typ, _ := apiutil.GetAnnotationValue(route.Annotations, "server", "type")

	handler = getHandlerName(handler)
	var reqBody string
	if len(route.RequestType.Name) > 0 {
		var bodyBuilder strings.Builder
		t := template.Must(template.New("parseRequest").Parse(parseRequestTemplate))
		if err := t.Execute(&bodyBuilder, map[string]string{
			"requestType": typesPacket + "." + util.Title(route.RequestType.Name),
		}); err != nil {
			return err
		}
		reqBody = bodyBuilder.String()
	}

	var req string
	switch typ {
	case SERVER_TYPE_HTML:
		if route.RequestType.Name == "" {
			req = "w,r"
		} else {
			req = "w,r,req"
		}
	default:
		if route.RequestType.Name == "" {
			req = ""
		} else {
			req = "req"
		}
	}
	var logicResponse string
	var writeResponse string
	var respWriter = `httpx.WriteJson(w, http.StatusOK, resp)`
	if len(route.ResponseType.Name) > 0 {
		logicResponse = "resp, err :="
		writeResponse = "resp, err"
	} else {
		logicResponse = "err :="
		writeResponse = "nil, err"
		respWriter = `httpx.Ok(w)`
	}
	var logicBodyBuilder strings.Builder
	switch typ {
	case SERVER_TYPE_HTML:
		t := template.Must(template.New("hasRespTemplate").Parse(hasRespTemplate_HtmlMode))
		if err := t.Execute(&logicBodyBuilder, map[string]string{
			"logic":         "New" + strings.TrimSuffix(strings.Title(handler), "Handler") + "Logic",
			"callee":        strings.Title(strings.TrimSuffix(handler, "Handler")),
			"req":           req,
			"logicResponse": logicResponse,
			"writeResponse": writeResponse,
			"respWriter":    respWriter,
		}); err != nil {
			return err
		}
	default:
		t := template.Must(template.New("hasRespTemplate").Parse(hasRespTemplate))
		if err := t.Execute(&logicBodyBuilder, map[string]string{
			"logic":         "New" + strings.TrimSuffix(strings.Title(handler), "Handler") + "Logic",
			"callee":        strings.Title(strings.TrimSuffix(handler, "Handler")),
			"req":           req,
			"logicResponse": logicResponse,
			"writeResponse": writeResponse,
			"respWriter":    respWriter,
		}); err != nil {
			return err
		}
	}
	respBody := logicBodyBuilder.String()

	if !strings.HasSuffix(handler, "Handler") {
		handler = handler + "Handler"
	}

	var bodyBuilder strings.Builder
	bodyTemplate := template.Must(template.New("handlerBodyTemplate").Parse(handlerBodyTemplate))
	if err := bodyTemplate.Execute(&bodyBuilder, map[string]string{
		"parseRequest": reqBody,
		"processBody":  respBody,
	}); err != nil {
		return err
	}
	return doGenToFile(dir, handler, group, route, bodyBuilder)
}

func doGenToFile(dir, handler string, group spec.Group, route spec.Route, bodyBuilder strings.Builder) error {
	if getHandlerFolderPath(group, route) != handlerDir {
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
	fp, created, err := apiutil.MaybeCreateFile(dir, getHandlerFolderPath(group, route), filename)
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
		"importPackages": genHandlerImports(group, route, parentPkg),
		"handlerName":    handler,
		"handlerBody":    strings.TrimSpace(bodyBuilder.String()),
	})
	if err != nil {
		return nil
	}
	formatCode := formatCode(buffer.String())
	_, err = fp.WriteString(formatCode)
	return err
}

func genHandlers(dir, proto string, api *spec.ApiSpec) error {
	for _, group := range api.Service.Groups {
		for _, route := range group.Routes {
			if proto != "" {
				e := genHandlerProto(dir, group, route)
				if e != nil {
					log.Println(e)
					return e
				}
				continue
			}
			if err := genHandler(dir, group, route); err != nil {
				return err
			}
		}
	}

	return nil
}

func genHandlerImports(group spec.Group, route spec.Route, parentPkg string) string {
	var imports []string
	imports = append(imports, fmt.Sprintf("\"%s\"",
		util.JoinPackages(parentPkg, getLogicFolderPath(group, route))))
	imports = append(imports, fmt.Sprintf("\"%s\"", util.JoinPackages(parentPkg, contextDir)))
	if len(route.RequestType.Name) > 0 {
		imports = append(imports, fmt.Sprintf("\"%s\"\n", util.JoinPackages(parentPkg, typesDir)))
	}
	imports = append(imports, fmt.Sprintf("\"%s/rest/httpx\"", vars.ProjectOpenSourceUrl))

	return strings.Join(imports, "\n\t")
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
