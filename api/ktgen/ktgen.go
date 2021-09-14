package ktgen

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
	apiBaseTemplate = `package {{.}}

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.BufferedReader
import java.io.InputStreamReader
import java.io.OutputStreamWriter
import java.net.HttpURLConnection
import java.net.URL

const val SERVER = "http://localhost:8080"

suspend fun apiRequest(
    method: String,
    uri: String,
    body: String = "",
    onOk: ((String) -> Unit)? = null,
    onFail: ((String) -> Unit)? = null,
    eventually: (() -> Unit)? = null
) = withContext(Dispatchers.IO) {
    val url = URL(SERVER + uri)
    with(url.openConnection() as HttpURLConnection) {
        connectTimeout = 3000
        requestMethod = method
        doInput = true
        if (method == "POST" || method == "PUT" || method == "PATCH") {
            setRequestProperty("Content-Type", "application/json")
            doOutput = true
            val wr = OutputStreamWriter(outputStream)
            wr.write(body)
            wr.flush()
        }

         try {
            if (responseCode >= 400) {
                BufferedReader(InputStreamReader(errorStream)).use {
                    val response = it.readText()
                    onFail?.invoke(response)
                }
                return@with
            }
            //response
            BufferedReader(InputStreamReader(inputStream)).use {
                val response = it.readText()
                onOk?.invoke(response)
            }
        } catch (e: Exception) {
            e.message?.let { onFail?.invoke(it) }
        }
    }
    eventually?.invoke()
}
`
	apiTemplate = `package {{with .Info}}{{.Desc}}{{end}}

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.Serializable

object {{with .Info}}{{.Title}}{{end}}{
	{{range .Types}}
	@Serializable
	{{if eq 0 (len .Members)}}class {{.Name}} {} {{else}}data class {{.Name}}({{$length := (len .Members)}}{{range $i,$item := .Members}}
		val {{with $item}}{{lowCamelCase .Name}}: {{toKtType .Type}}{{end}}{{if ne $i (add $length -1)}},{{end}}{{end}}
	){{end}}{{end}}
	{{with .Service}}
	{{range .Routes}}suspend fun {{routeToFuncName .Method .Path}}({{with .RequestType}}{{if ne .Name ""}}
		req:{{.Name}},{{end}}{{end}}
		onOk: (({{with .ResponseType}}{{.Name}}{{end}}) -> Unit)? = null,
        onFail: ((String) -> Unit)? = null,
        eventually: (() -> Unit)? = null
    ){
        apiRequest("{{upperCase .Method}}","{{.Path}}",{{with .RequestType}}{{if ne .Name ""}}body=Json.encodeToString(req),{{end}}{{end}} onOk = { {{with .ResponseType}}
            onOk?.invoke({{if ne .Name ""}}Json.decodeFromString(it){{end}}){{end}}
        }, onFail = onFail, eventually =eventually)
    }
	{{end}}{{end}}
}`
)

func genBase(dir, pkg string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}
	path := filepath.Join(dir, "Base.kt")
	if _, e := os.Stat(path); e == nil {
		log.Println("Base.kt already exists, skipped it.")
		return nil
	}

	file, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	t, e := template.New("n").Parse(apiBaseTemplate)
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
	path := filepath.Join(dir, name+".kt")
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

	t, e := template.New("api").Funcs(util.FuncsMap).Parse(apiTemplate)
	if e != nil {
		return e
	}
	return t.Execute(file, api)
}
