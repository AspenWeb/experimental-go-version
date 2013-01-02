package goaspen

import (
	"strings"
	"text/template"
)

var (
	simplateTmplCommonHeader = `
package {{.GenPackage}}
// GENERATED FILE - DO NOT EDIT
// Rebuild with goaspen-build!

import (
    "net/http"

    "github.com/meatballhat/goaspen"
)

`
	simplateTypeRenderedTmpl = `
import (
    "bytes"
    "text/template"
)

{{.InitPage.Body}}

var (
    simplateTmplMap{{.FuncName}} = map[string]*template.Template{
        {{range .TemplatePages}}
        "{{.Spec.ContentType}}": template.Must(template.New("{{.Parent.FuncName}}!{{.Spec.ContentType}}").Parse(__BACKTICK__{{.Body}}__BACKTICK__)),
        {{end}}
    }
    local{{.FuncName}}App = goaspen.DeclareApp("{{.GenPackage}}")
    _ = local{{.FuncName}}App.NewHandlerFuncRegistration("/{{.Filename}}", SimplateHandlerFunc{{.FuncName}})
)

func SimplateHandlerFunc{{.FuncName}}(w http.ResponseWriter, req *http.Request) {
    var err error
    app := local{{.FuncName}}App
    ctx := make(map[string]interface{})
    response := app.NewHTTPResponseWrapper(w, req)

    {{.LogicPage.Body}}

    {{range .TemplatePages}}
    response.RegisterContentTypeHandler("{{.Spec.ContentType}}",
        func(response *goaspen.HTTPResponseWrapper) {
            tmpl := simplateTmplMap{{.Parent.FuncName}}["{{.Spec.ContentType}}"]
            var tmplBuf bytes.Buffer

            err = tmpl.Execute(&tmplBuf, ctx)
            if err != nil {
                response.SetError(err)
                return
            }

            response.SetContentType("{{.Spec.ContentType}}")
            response.SetBodyBytes(tmplBuf.Bytes())
        })
    {{end}}

    response.NegotiateAndCallHandler()
    if err != nil {
        response.SetError(err)
    }

    response.Respond()
}
`
	simplateTypeJSONTmpl = `
{{.InitPage.Body}}

var (
    local{{.FuncName}}App = goaspen.DeclareApp("{{.GenPackage}}")
    _ = local{{.FuncName}}App.NewHandlerFuncRegistration("/{{.Filename}}", SimplateHandlerFunc{{.FuncName}})
)

func SimplateHandlerFunc{{.FuncName}}(w http.ResponseWriter, req *http.Request) {
    var err error
    app := local{{.FuncName}}App
    ctx := make(map[string]interface{})
    response := app.NewHTTPResponseWrapper(w, req)

    response.RegisterContentTypeHandler("{{.ContentType}}",
        func(response *goaspen.HTTPResponseWrapper) {
        {{.LogicPage.Body}}
        })

    response.NegotiateAndCallHandler()
    if err != nil {
        response.SetError(err)
    }

    response.RespondJSON()
}
`
	simplateTypeNegotiatedTmpl = simplateTypeRenderedTmpl
)

func escapedSimplateTemplate(tmplString, name string) *template.Template {
	tmpl := template.New(name)
	tmplString = simplateTmplCommonHeader + tmplString
	escTmplString := strings.Replace(tmplString, "__BACKTICK__", "`", -1)
	return template.Must(tmpl.Parse(escTmplString))
}
