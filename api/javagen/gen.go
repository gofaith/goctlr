package javagen

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gofaith/goctl/api/spec"
	"github.com/gofaith/goctl/api/util"
	"github.com/iancoleman/strcase"
)

const (
	apiBaseTemplate = `package {{.}};

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.OutputStreamWriter;
import java.net.HttpURLConnection;
import java.net.URL;

public class Base {
	private static final String SERVER = "http://localhost:8080";
	public static String request(String method, String uri,String body)throws Exception{
		URL url = new URL(SERVER + uri);
		HttpURLConnection connection = (HttpURLConnection) url.openConnection();
		connection.setConnectTimeout(3000);
		connection.setRequestMethod(method);
		connection.setDoInput(true);

		switch (method) {
			case "POST":
			case "PUT":
			case "PATCH":
				connection.setRequestProperty("Content-Type", "application/json");
				connection.setDoOutput(true);
				OutputStreamWriter writer = new OutputStreamWriter(connection.getOutputStream());
				writer.write(body);
				writer.close();
		}

		BufferedReader br = new BufferedReader(new InputStreamReader(connection.getErrorStream()));
		StringBuffer buffer = new StringBuffer();
		int i;
		while ((i = br.read()) != -1) {
			buffer.append((char) i);
		}
		br.close();

		if (connection.getResponseCode() >=400) {
			throw new Exception(buffer.toString());
		}
		return buffer.toString();
	}
}`
	apiTemplate=`package {{.Info.Desc}};

import com.google.gson.Gson;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class {{.Info.Title}} {
	{{range .Types}}
	public statis class {{.Name}} { {{range .Members}}
		public {{toJavaType .Type}} {{lowCamelCase .Name}};{{end}}
	}{{end}}
	{{with .Service}}{{range .Routes}}
	public static {{with .ResponseType}}{{if eq .Name ""}}void{{else}}{{.Name}}{{end}} {{routeToFuncName .Method .Path}}({{with .RequestType}}{{if ne .Name ""}}{{.Name}} request{{else}}{{end}}{{end}}) throws Exception {
		String res = Base.request("{{.Method}}", "{{.Path}}", {{with .RequestType}}{{if ne .Name ""}}new Gson().toJson(request){{else}}null{{end}}{{end}});
		{{with .ResponseType}}{{if ne .Name ""}}return new Gson().fromJson(res, Response.class);{{end}}{{end}}
	}{{end}}{{end}}
}
`
)

func genBase(dir, pkg string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}
	path := filepath.Join(dir, "Base.java")
	if _, e := os.Stat(path); e == nil {
		fmt.Println("Base.java already exists. Skipped it.")
		return nil
	}

	file, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	api.Info.Desc = pkg
	t, e := template.New("Base.java").Parse(apiBaseTemplate)
	if e != nil {
		return e
	}
	e = t.Execute(file, pkg)
	if e != nil {
		return e
	}
	return nil
}

func genApi(dir, pkg string, api *spec.ApiSpec) error {
	name := strcase.ToCamel(api.Info.Title + "Api")
	path := filepath.Join(dir, name+".java")
	api.Info.Title = name
	api.Info.Desc = pkg
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}

	file, e := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	t, e := template.New(name).Funcs(util.FuncsMap).Parse(apiTemplate)
	if e != nil {
		return e
	}
	return t.Execute(file, api)
}
