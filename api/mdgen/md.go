package mdgen

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"

	"github.com/gofaith/go-zero/core/stringx"
	"github.com/gofaith/goctl/api/gogen"
	"github.com/gofaith/goctl/api/spec"
	"github.com/gofaith/goctl/api/util"
)

const (
	markdownTemplate = `
### {{.index}}. {{.routeComment}}

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

	var builder strings.Builder
	builder.WriteString("# " + api.Info.Desc)
	for index, route := range api.Service.Routes {
		routeComment, _ := util.GetAnnotationValue(route.Annotations, "doc", "summary")
		if len(routeComment) == 0 {
			routeComment = "N/A"
		}

		requestContent, responseContent, e := requestAndResponseBody(api, route)
		if e != nil {
			return e
		}

		t := template.Must(template.New("markdownTemplate").Parse(markdownTemplate))
		var tmplBytes bytes.Buffer
		err := t.Execute(&tmplBytes, map[string]string{
			"index":           strconv.Itoa(index + 1),
			"routeComment":    routeComment,
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
	_, e = f.WriteString(strings.Replace(builder.String(), "&#34;", `"`, -1))
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
