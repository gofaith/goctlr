package jsgen

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
	baseTemplate = `
var server='http://localhost:8888'
function apiRequest(method,uri,body,onOk,onFail,eventually,headers,onProgress){
    var xhr=new XMLHttpRequest();
    xhr.onreadystatechange=function(e){
        if(xhr.readyState==4){
            if(xhr.status==200){
                if(onOk){
					if(xhr.responseText){
						onOk(JSON.parse(xhr.responseText));
					}else{
						onOk();
					}
                }
            }else {
                if(onFail){
                    try{
                        onFail(JSON.parse(xhr.responseText))
                    }catch(e){
                        onFail(xhr.responseText)
                    }
                }
            }
            if(eventually){
                eventually()
            }
        }
    }
    xhr.open(method,server+uri,true)
	xhr.setRequestHeader('Cookies',document.cookie)
	if(headers){
		for(key in headers){
			xhr.setRequestHeader(key,headers[key]);
		}
	}

    //progress
    if (onProgress){
        xhr.upload.addEventListener('progress',function(ev){
            if(ev.lengthComputable){
                let percent=Math.round(ev.loaded*100/ev.total);
                onProgress(percent,ev.loaded,ev.total);
            }
        })
    }

    if(body){
        if (typeof body == 'string'){
			xhr.setRequestHeader('Content-Type','application/json')
			xhr.send(body)
		}else if(body instanceof File||body instanceof Blob){
			xhr.setRequestHeader('Content-Type',body.type)
			xhr.send(body)
		}else if(body instanceof FormData){
			xhr.setRequestHeader('Content-Type','multipart/form-data')
			xhr.send(body)
        }else{
			xhr.setRequestHeader('Content-Type','application/json')
            xhr.send(JSON.stringify(body))
        }
    }else{
        xhr.send()
    }
}`
	apiTemplate = `{{with .Service}}{{range .Routes}}
//{{.Summary}}
function {{routeToFuncName .Method .Path}}(req,onOk,onFail,eventually,headers,onProgress){
    apiRequest('{{upperCase .Method}}','{{.Path}}',req,onOk,onFail,eventually,headers,onProgress)
}{{end}}{{end}}`
)

func genBase(dir string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}
	path := filepath.Join(dir, "base.js")
	if _, e := os.Stat(path); e == nil {
		log.Println("base.js already exists , skipped it.")
		return nil
	}

	file, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	_, e = file.WriteString(baseTemplate)
	return e
}

func genApi(dir string, api *spec.ApiSpec) error {
	name := strcase.ToSnake(api.Info.Title + "_api")
	path := filepath.Join(dir, name+".js")
	api.Info.Title = name

	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}

	file, e := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	t, e := template.New("api").Funcs(util.FuncsMap).Parse(apiTemplate)
	if e != nil {
		return e
	}
	return t.Execute(file, api)
}
