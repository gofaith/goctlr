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


Future apiRequest(String method, String uri, dynamic body, Function(String)? onOk, Function(ErrorCode)? onFail, Function()? eventually) async {
	final sp = await SharedPreferences.getInstance();
	BaseRequest req = Request(method, Uri.parse(await _getServer(sp) + uri));
	final token = sp.getString('token') ?? '';
  
	try {
	  if (body is MultipartRequest) {
		// Multipart
		final r = MultipartRequest(method, Uri.parse(await _getServer(sp) + uri));
		if (token.isNotEmpty) {
		  r.headers['Authorization'] = token;
		}
		body.fields.forEach((key, value) {
		  r.fields[key] = value;
		});
		r.files.addAll(body.files);
		req = r;
	  } else if (body is File) {
		// File
		final r = Request(method, Uri.parse(await _getServer(sp) + uri));
		r.headers['Content-Type'] = 'application/octet-stream';
		if (token.isNotEmpty) {
		  r.headers['Authorization'] = token;
		}
		final fi = body;
		r.bodyBytes = List.from(fi.readAsBytesSync());
		req = r;
	  } else if (body != null) {
		// Json
		final r = Request(method, Uri.parse(await _getServer(sp) + uri));
		r.headers['Content-Type'] = 'application/json; charset=utf-8';
		if (token.isNotEmpty) {
		  r.headers['Authorization'] = token;
		}
		r.body = jsonEncode(body);
		req = r;
	  } else {
		if (token.isNotEmpty) {
		  req.headers['Authorization'] = token;
		}
	  }
  
	  final res = await req.send();
	  final str = await res.stream.bytesToString();
	  if (res.statusCode == 200) {
		onOk?.call(str);
	  } else {
		try {
		  onFail?.call(ErrorCode.fromJson(jsonDecode(str)));
		} catch (e) {
		  onFail?.call(ErrorCode(code: res.statusCode, desc: '${res.statusCode}:$str'));
		}
	  }
	} catch (e, stack) {
	  // ignore: avoid_print
	  print(stack);
	  updateServer(sp);
	  onFail?.call(ErrorCode(desc: e.toString()));
	}
	eventually?.call();
  }
  
  const _serverKey = '--server--';
  Future<String> _getServer(SharedPreferences sp) async {
	final s = sp.getString(_serverKey) ?? '';
	if (s.isEmpty) {
	  return await updateServer(sp);
	}
	return s;
  }
  
  Future<String> updateServer(SharedPreferences sp) async {
	final results = await DnsUtils.lookupRecord('_server.example.com', RRecordType.TXT);
	if (results != null && results.isNotEmpty) {
	  final s = results[0].data;
	  sp.setString(_serverKey, s);
	  if (kDebugMode) {
		print('server updated:$s');
	  }
	  return s;
	}
	return '';
  }
`
	apiApiTemplate = `import 'dart:convert';
import './base.dart';
import 'package:json_annotation/json_annotation.dart';

part '{{snakeCase .Info.Title}}.g.dart';

{{range .Types}}
@JsonSerializable()
class {{.Name}} {
	{{range .Members}}
	/// {{.Comment}}
	{{toDartType .Type}} {{lowCamelCase .Name}};{{end}}
	{{.Name}}({{if ne 0 (len .Members)}}{ {{range .Members}}
		this.{{lowCamelCase .Name}}{{if ne (dartDefaultValue .Type) ""}} = {{dartDefaultValue .Type}}{{end}},{{end}}
	}{{end}});
	factory {{.Name}}.fromJson(Map<String, dynamic> jsonObject) => _${{.Name}}FromJson(jsonObject);
	Map<String, dynamic> toJson() => _${{.Name}}ToJson(this);
}
{{end}}

class {{with .Info}}{{.Title}}{{end}} { {{with .Service}}{{range .Routes}}
	static Future {{routeToFuncName .Method .Path}}(
		{{with .RequestType}}{{if ne .Name ""}}{{.Name}}{{else}}dynamic{{end}} req,{{end}}
		{Function({{if ne .ResponseType.Name ""}}{{.ResponseType.Name}} res{{end}})? onOk,
		Function(ErrorCode e)? onFail,
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
