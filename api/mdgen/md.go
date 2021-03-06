package mdgen

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/gofaith/go-zero/core/stringx"
	"github.com/gofaith/goctlr/api/gogen"
	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/api/util"
)

const (
	markdownTemplateHead = `{{with .Info}}# {{.Desc}}

编辑者：{{.Author}}

联系邮箱：{{.Email}}
{{end}}

{{with .Service}}{{range .Groups}}
- {{if .Jwt}}👥{{end}}{{.Desc}}{{range .Routes}}
	- [{{.Summary}}](#{{.Summary}}){{end}}
{{end}}
{{end}}	`
	markdownTemplate = `
### {{.routeComment}}


{{.routeDesc}}

路由： ` + "`" + `{{.uri}}` + "`" + `

方法： ` + "`" + `{{.method}}` + "`" + `

请求体：
{{.requestContent}}

响应体：

{{.responseContent}}  

`
)

func genMd(api *spec.ApiSpec, path string) error {
	f, e := os.OpenFile(path+".md", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer f.Close()

	var builder = new(strings.Builder)
	e = template.Must(template.New("markdownTemplateHead").Parse(markdownTemplateHead)).Execute(builder, api)
	if e != nil {
		return e
	}

	for index, route := range api.Service.Routes {
		routeComment, _ := util.GetAnnotationValue(route.Annotations, "doc", "summary")
		if len(routeComment) == 0 {
			routeComment = "N/A"
		}

		routeDesc, _ := util.GetAnnotationValue(route.Annotations, "doc", "desc")

		requestContent, responseContent, e := requestAndResponseBody(api, route)
		if e != nil {
			return e
		}

		t := template.Must(template.New("markdownTemplate").Parse(markdownTemplate))
		var tmplBytes bytes.Buffer
		err := t.Execute(&tmplBytes, map[string]string{
			"index":           strconv.Itoa(index + 1),
			"routeComment":    routeComment,
			"routeDesc":       routeDesc,
			"method":          strings.ToUpper(route.Method),
			"uri":             route.Path,
			"requestType":     "`" + stringx.TakeOne(route.RequestType.Name, "-") + "`",
			"responseType":    "`" + stringx.TakeOne(route.ResponseType.Name, "-") + "`",
			"requestContent":  requestContent,
			"responseContent": responseContent,
		})
		if err != nil {
			return err
		}

		builder.Write(tmplBytes.Bytes())
	}
	s := strings.Replace(builder.String(), "&#34;", `"`, -1)
	s = strings.ReplaceAll(s, "&#43;", "+")
	_, e = f.WriteString(s)
	return e
}

func requestAndResponseBody(api *spec.ApiSpec, route spec.Route) (string, string, error) {
	rts, rpts := util.GetAllTypes(api, route)
	r, err := gogen.BuildTypes(rts)
	if err != nil {
		return "", "", err
	}
	rp, e := gogen.BuildTypes(rpts)
	if e != nil {
		return "", "", e
	}
	return fmt.Sprintf("```go\n%s\n```", r), fmt.Sprintf("```go\n%s\n```", rp), nil
}
