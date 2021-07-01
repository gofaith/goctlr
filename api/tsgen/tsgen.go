package tsgen

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
	apiBaseTemplate = `import pako from 'pako';

export class ErrorCode{
	public code:number;
	public desc:string;
	constructor(code:number,desc:string){
		this.code=code;
		this.desc=desc;
	}
}

export function apiRequest(method: string, uri: string, body: any, onOk: (res: string) => void, onFail: (e: ErrorCode) => void, eventually?: () => void, headers?: Record<string, string>) {
	const xhr = new XMLHttpRequest();
	xhr.onreadystatechange = function (ev: Event) {
		if (xhr.readyState != 4) {
			return;
		}
		if (xhr.status == 200) {
			onOk(xhr.responseText);
		}else if(xhr.status==401){
			doLogout();
		} else {
			try{
				let err:ErrorCode=JSON.parse(xhr.responseText)
				if (err.code==4){
					doLogout();
				}else{
					onFail(err)
				}
			}catch(e){
				onFail(new ErrorCode(1,e))
			}
		}
		if (eventually) {
			eventually();
		}
	}
	xhr.open(method, 'http://localhost:8080' + uri, true);
	if (headers) {
		for (let key in headers) {
			xhr.setRequestHeader(key, headers[key]);
		}
	}
	if (body) {
		if (typeof body == 'string') {
			xhr.setRequestHeader('Content-Type', 'application/json')
			xhr.setRequestHeader('Content-Encoding', 'gzip')
			xhr.send(pako.gzip(body, { to: 'string' }))
		} else if (body instanceof File || body instanceof Blob) {
			xhr.setRequestHeader('Content-Type', body.type)
			xhr.send(body)
		} else {
			xhr.setRequestHeader('Content-Type', 'application/json')
			xhr.setRequestHeader('Content-Encoding', 'gzip')
			xhr.send(pako.gzip(JSON.stringify(body), { to: 'string' }))
		}
	} else {
		xhr.send()
	}
}`

	apiTemplate = `import {apiRequest, ErrorCode} from "./apiRequest"

export class {{with .Info}}{{.Title}}{{end}} { {{with .Service}}{{range .Routes}}
	/** {{.Summary}}{{if ne .Desc ""}}
	{{.Desc}}{{end}}*/
	static {{routeToFuncName .Method .Path}}({{with .RequestType}}{{if ne .Name ""}}
		req:{{.Name}},{{end}}{{end}}
		onOk: ({{with .ResponseType}}{{if ne .Name ""}}res: {{.Name}}{{end}}{{end}}) => void, 
		onFail: (e: ErrorCode) => void, 
		eventually?: () => void, 
		headers?: Record<string, string>
	) {
        apiRequest('{{upperCase .Method}}', '{{.Path}}', {{with .RequestType}}{{if ne .Name ""}}req{{else}}null{{end}}{{end}}, res=>{
            onOk({{with .ResponseType}}{{if ne .Name ""}}{{.Name}}.fromJson(JSON.parse(res)){{end}}{{end}})
        }, onFail, eventually, headers);
	}{{end}}{{end}}
}
{{range .Types}}
export class {{.Name}} { {{range .Members}}
	public {{tagGet .Tag "json"}}: {{toTsType .Type}};	//{{tagTail .Tag "json"}}，{{.Comment}} {{end}}
	constructor() { {{range .Members}}
		this.{{tagGet .Tag "json"}} = {{tsDefaultValue .Type}};{{end}}
	}
	static fromJson(json: any): {{.Name}} {
		const obj = new {{.Name}}();
		{{range .Members}}
		obj.{{tagGet .Tag "json"}} = json['{{tagGet .Tag "json"}}'];{{end}}
		return obj;
	}
}{{end}}
`
)

func genBase(dir string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		log.Println(e)
		return e
	}

	path := filepath.Join(dir, "apiRequest.ts")
	if _, e := os.Stat(path); e == nil {
		log.Println("apiRequest.ts already exists, skipped it.")
		return nil
	}

	file, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		log.Println(e)
		return e
	}
	defer file.Close()

	t, e := template.New("apiRequest.ts").Parse(apiBaseTemplate)
	if e != nil {
		log.Println(e)
		return e
	}
	e = t.Execute(file, nil)
	if e != nil {
		log.Println(e)
		return e
	}
	return nil
}

func genApi(dir string, api *spec.ApiSpec) error {
	name := strcase.ToCamel(api.Info.Title + "Api")
	path := filepath.Join(dir, name+".ts")
	api.Info.Title = name

	e := os.MkdirAll(dir, 0755)
	if e != nil {
		log.Println(e)
		return e
	}

	file, e := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if e != nil {
		log.Println(e)
		return e
	}
	defer file.Close()

	t, e := template.New(name).Funcs(util.FuncsMap).Parse(apiTemplate)
	if e != nil {
		log.Println(e)
		return e
	}
	return t.Execute(file, api)
}
