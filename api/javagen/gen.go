package javagen

import (
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/api/util"
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
	apiTemplate = `package {{with .Info}}{{.Desc}}{{end}};

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;
import org.json.JSONTokener;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class {{with .Info}}{{.Title}}{{end}} {
	{{range .Types}}
	public static class {{.Name}} extends JSONObject{ {{range .Members}}
		public {{toJavaPrimitiveType .Type}} {{lowCamelCase .Name}};{{end}}
		@Override
		public String toString(){
			try { {{range .Members}}
				{{if isJavaTypeNullable .Type}}if (this.{{lowCamelCase .Name}} == null) {
					put("{{tagGet .Tag "json"}}", {{if eq .Type "string"}}""{{else}}JSONObject.NULL{{end}});
				}else{
					{{end}}{{if isAtomicType .Type}}put("{{tagGet .Tag "json"}}",this.{{lowCamelCase .Name}});{{else if isListType .Type}}JSONArray {{lowCamelCase .Name}}JsonArray = new JSONArray();
					for (int i = 0; i < this.{{lowCamelCase .Name}}.size(); i++) {
						{{lowCamelCase .Name}}JsonArray.put(this.{{lowCamelCase .Name}}.get(i));
					}
					put("{{tagGet .Tag "json"}}", {{lowCamelCase .Name}}JsonArray);{{else}}put("{{tagGet .Tag "json"}}",this.{{lowCamelCase .Name}});{{end}}
				{{if isJavaTypeNullable .Type}}}{{end}}{{end}}
			} catch (JSONException e) {
				e.printStackTrace();
			}
			return super.toString();
		}
		public static {{.Name}} fromJson(JSONObject object) {
			{{.Name}} v = new {{.Name}}();
			try { {{range .Members}}
				{{if isAtomicType .Type}}v.{{lowCamelCase .Name}} = object.{{toJavaGetFunc .Type}}("{{tagGet .Tag "json"}}");{{else if isListType .Type}}if (object.has("{{tagGet .Tag "json"}}")&&!object.isNull("{{tagGet .Tag "json"}}")) {
					v.{{lowCamelCase .Name}} = new ArrayList<>();
					JSONArray {{lowCamelCase .Name}}JsonArray = object.getJSONArray("{{tagGet .Tag "json"}}");
					for (int i = 0; i < {{lowCamelCase .Name}}JsonArray.length(); i++) {
						v.{{lowCamelCase .Name}}.add({{if isClassListType .Type}}{{getCoreType .Type}}.fromJson({{lowCamelCase .Name}}JsonArray.getJSONObject(i)){{else}}{{lowCamelCase .Name}}JsonArray.{{toJavaGetFunc (getCoreType .Type)}}(i){{end}});
					}
				}{{else}}v.{{lowCamelCase .Name}} = {{.Type}}.fromJson(object.getJSONObject("{{tagGet .Tag "json"}}"));{{end}}{{end}}
			} catch (JSONException e) {
				e.printStackTrace();
			}
			return v;
		}
	}{{end}}
	{{with .Service}}{{range .Routes}}
	public static {{with .ResponseType}}{{if eq .Name ""}}void{{else}}{{.Name}}{{end}}{{end}} {{routeToFuncName .Method .Path}}({{with .RequestType}}{{if ne .Name ""}}{{.Name}} request{{else}}{{end}}{{end}}) throws Exception {
		{{with .ResponseType}}{{if ne .Name ""}}String res = {{end}}{{end}}Base.request("{{upperCase .Method}}", "{{.Path}}", {{with .RequestType}}{{if ne .Name ""}}request.toString(){{else}}null{{end}}{{end}});{{with .ResponseType}}{{if ne .Name ""}}
		return {{.Name}}.fromJson((JSONObject) new JSONTokener(res).nextValue());{{end}}{{end}}
	} {{end}}{{end}}
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
		log.Println("Base.java already exists. Skipped it.")
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
