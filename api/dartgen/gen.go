package dartgen

import (
	"errors"
	"log"
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

class ErrorCode {
	int code;
	String desc;
	ErrorCode({
	this.code = 0,
	this.desc = '',
	});
	factory ErrorCode.fromJson(Map<String, dynamic> jsonObject) => ErrorCode(
		code: jsonObject['code'],
		desc: jsonObject['desc'],
		);
	Map<String, dynamic> toJson() => {
		'code': code,
		'desc': desc,
		};
}

const server = 'https://api.example.com';

Future apiRequest(String method, String uri, dynamic body, Function(String)? onOk, Function(ErrorCode)? onFail, Function()? eventually) async {
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
	onFail?.call(ErrorCode(desc: e.toString()));
	}, onDone: () {
	completer.complete(buf.toString());
	}, cancelOnError: true);

	final str = await completer.future;
	if (res.statusCode == 200) {
	onOk?.call(str);
	} else {
	try {
		onFail?.call(ErrorCode.fromJson(jsonDecode(str)));
	} catch (e) {
		onFail?.call(ErrorCode(code: -1, desc: '${res.statusCode}:$str'));
	}
	}

	eventually?.call();
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
		this.{{lowCamelCase .Name}} = {{dartDefaultValue .Type}},{{end}}
	}{{end}});
	factory {{.Name}}.fromJson(Map<String, dynamic> jsonObject) => {{.Name}}({{range .Members}}
		{{lowCamelCase .Name}}: {{if isDirectType .Type}}jsonObject['{{tagGet .Tag "json"}}']{{else if isClassListType .Type}}(jsonObject['{{tagGet .Tag "json"}}'] as List<dynamic>)
			.map((i)=>{{getCoreType .Type}}.fromJson(i))
			.toList(){{else}}{{.Type}}.fromJson(jsonObject['{{tagGet .Tag "json"}}']){{end}},{{end}}
	);
	Map<String, dynamic> toJson() => { {{range .Members}}
		'{{tagGet .Tag "json"}}': {{if isDirectType .Type}}{{lowCamelCase .Name}}{{else if isClassListType .Type}}{{lowCamelCase .Name}}.map((i) => i.toJson()).toList(){{else}}{{lowCamelCase .Name}}.toJson(){{end}},{{end}}
	};
}
{{end}}

class {{with .Info}}{{.Title}}{{end}} { {{with .Service}}{{range .Routes}}
	static Future {{routeToFuncName .Method .Path}}(
		{{with .RequestType}}{{if ne .Name ""}}{{.Name}}{{else}}dynamic{{end}} req,{{end}}
		{Function({{with .ResponseType}}{{.Name}}{{end}})? onOk,
		Function(ErrorCode)? onFail,
		Function()? eventually}
	) async {
		await apiRequest('{{upperCase .Method}}', '{{.Path}}',req,(data){
			if (onOk != null){ {{with .ResponseType}}{{if ne .Name ""}}
				final res = {{.Name}}.fromJson(jsonDecode(data));
				onOk(res);{{else}}
				onOk();{{end}}
			}{{end}}
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
		log.Println("base.dart already exists, skipped it.")
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
	api.Info.Title = strcase.ToCamel(api.Info.Title + "Api")
	path := filepath.Join(dir, strcase.ToSnake(api.Info.Title)+".dart")

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
