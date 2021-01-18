package dartgen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gofaith/go-zero/core/logx"
	"github.com/gofaith/goctlr/api/parser"
	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/api/util"
	"github.com/iancoleman/strcase"
	"github.com/urfave/cli"
)

const (
	apiBaseTemplate = `import 'dart:async';
import 'dart:convert';
import 'dart:io';

final server = 'http://localhost';

Future apiRequest(
	String method,
	String uri,
	dynamic body,
	Function(String) onOk,
	Function(String) onFail,
	Function() eventually) async {
	final client = HttpClient();
	final req = await client.openUrl(method, Uri.parse(server + uri));
	req.headers.add('Content-Type', 'application/json');
	if (body != null) {
	req.write(jsonEncode(body));
	}

	final res = await req.close();

	final buf = StringBuffer();
	final completer = Completer<String>();
	res.transform(utf8.decoder).listen((data) {
	buf.write(data);
	}, onError: (e) {
	print(e);
	}, onDone: () {
	completer.complete(buf.toString());
	}, cancelOnError: true);

	final str = await completer.future;
	if (res.statusCode == 200) {
	if (onOk != null) {
		print('ok');
		onOk(str);
	}
	} else {
	try {
		Map<String, dynamic> e = jsonDecode(str);
		if (onFail != null) {
		onFail(e['desc']);
		}
	} catch (e) {
		if (onFail != null) {
		onFail('${res.statusCode}:$str');
		}
	}
	}

	if (eventually != null) {
	eventually();
	}
}
	
`
	apiApiTemplate = `import 'dart:convert';
import './base.dart';
{{range .Types}}
class {{.Name}} {
	{{range .Members}}
	/// {{.Comment}}
	{{toDartType .Type}} {{lowCamelCase .Name}};{{end}}
	{{.Name}}({{if ne 0 (len .Members)}}{ {{range .Members}}
		this.{{lowCamelCase .Name}},{{end}}
	}{{end}});
	factory {{.Name}}.fromJson(Map<String, dynamic> jsonObject) {
		return {{.Name}}({{range .Members}}
			{{lowCamelCase .Name}}: {{if isDirectType .Type}}jsonObject['{{tagGet .Tag "json"}}']{{else if isClassListType .Type}}(jsonObject['{{tagGet .Tag "json"}}'] as List<dynamic>)
				.map((i)=>{{getCoreType .Type}}.fromJson(i))
				.toList(){{else}}{{.Type}}.fromJson(jsonObject['{{tagGet .Tag "json"}}']){{end}},{{end}}
		);
	}
	Map<String, dynamic> toJson() {
		return { {{range .Members}}
			'{{tagGet .Tag "json"}}': {{if isDirectType .Type}}{{lowCamelCase .Name}}{{else if isClassListType .Type}}{{lowCamelCase .Name}}.map((i) => i.toJson()).toList(){{else}}{{lowCamelCase .Name}}.toJson(){{end}},{{end}}
		};
	}
}
{{end}}

class {{with .Info}}{{.Title}}{{end}} { {{with .Service}}{{range .Routes}}
	static Future {{routeToFuncName .Method .Path}}(
		{{with .RequestType}}{{if ne .Name ""}}{{.Name}}{{else}}dynamic{{end}} req,{{end}}
		{Function({{with .ResponseType}}{{.Name}}{{end}}) onOk,
		Function(String) onFail,
		Function() eventually}
	) async {
		await apiRequest('{{upperCase .Method}}', '{{.Path}}',req,(data){
			{{with .ResponseType}}{{if ne .Name ""}}
			if (onOk != null){
				final res = {{.Name}}.fromJson(jsonDecode(data));
				onOk(res);
			}
			{{end}}{{end}}
		},onFail,eventually);
	}{{end}}
}
{{end}}`
)

func genBase(dir string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}
	path := filepath.Join(dir, "base.dart")
	if _, e := os.Stat(path); e == nil {
		fmt.Println("base.dart already exists, skipped it.")
		return nil
	}

	file, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	file.WriteString(apiBaseTemplate)
	return nil
}

func genApi(dir string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}
	name := strcase.ToCamel(api.Info.Title + "Api")
	path := filepath.Join(dir, name+".dart")
	api.Info.Title = name

	file, e := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	t, e := template.New("api").Funcs(util.FuncsMap).Parse(apiApiTemplate)
	if e != nil {
		return e
	}
	return t.Execute(file, api)
}

func DartCommand(c *cli.Context) error {
	apiFile := c.String("api")
	dir := c.String("dir")
	if len(apiFile) == 0 {
		return errors.New("missing -api")
	}
	if len(dir) == 0 {
		return errors.New("missing -dir")
	}

	p, err := parser.NewParser(apiFile)
	if err != nil {
		return err
	}
	api, err := p.Parse()
	if err != nil {
		return err
	}

	logx.Must(genBase(dir, api))
	logx.Must(genApi(dir, api))
	return nil
}
