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

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.features.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import java.io.ByteArrayInputStream
import java.io.ByteArrayOutputStream
import java.util.zip.GZIPInputStream
import java.util.zip.GZIPOutputStream

@Serializable
data class ErrorCode(
	val code:Int,
	val desc:String,
)

private val SERVER:String = "http://localhost:8888"
private val client= HttpClient(CIO){
	expectSuccess = false
	install(HttpTimeout){
		requestTimeoutMillis = 5000
	}
}

suspend fun apiRequest(
	method: String,
	uri: String,
	body: String = "{}",
	onOk: ((String) -> Unit)? = null,
	onFail: ((ErrorCode) -> Unit)? = null,
	eventually: (() -> Unit)? = null
) = withContext(Dispatchers.IO) {
	try {
		val response: HttpResponse = client.request(SERVER + uri) {
			this.method = HttpMethod.parse(method)
			header("Content-Type", "application/json")
			header("Accept-Encoding","gzip")

			if (body.length > 1024) {
				header("Content-Encoding", "gzip")
				val out = ByteArrayOutputStream()
				GZIPOutputStream(out).bufferedWriter().use { it.write(body) }
				this.body = out.toByteArray()
			}else{
				this.body = body
			}
		}

		val rp=if (response.headers["Content-Encoding"] == "gzip") {
			GZIPInputStream(ByteArrayInputStream(response.readBytes())).bufferedReader().use {
				it.readText()
			}
		}else{
			response.receive()
		}

		when (response.status.value) {
			200 -> {
				onOk?.invoke(rp)
			}
			400 -> {
				if (rp.startsWith("{")) {
					onFail?.invoke(Json.decodeFromString(rp))
				}else{
					onFail?.invoke(ErrorCode(0,"${response.status.value}:$rp"))
				}
			}
			else -> {
				onFail?.invoke(ErrorCode(0,"${response.status.value}:$rp"))
			}
		}
	} catch (e:Exception) {
		e.printStackTrace()
		onFail?.invoke(ErrorCode(0,"exception: ${e.localizedMessage?:""}"))
	}finally {
		eventually?.invoke()
	}
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
        onFail: ((ErrorCode) -> Unit)? = null,
        eventually: (() -> Unit)? = null
    ){
        apiRequest("{{upperCase .Method}}","{{.Path}}",{{with .RequestType}}{{if ne .Name ""}}body=Json.encodeToString(req),{{end}}{{end}} onOk = { {{with .ResponseType}}
            onOk?.invoke({{if ne .Name ""}}Json{ignoreUnknownKeys=true}.decodeFromString(it){{end}}){{end}}
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
