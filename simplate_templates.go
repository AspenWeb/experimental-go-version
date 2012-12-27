package goaspen

import (
	"strings"
	"text/template"
)

var (
	simplateTmplCommonHeader = `
package goaspen_gen
// GENERATED FILE - DO NOT EDIT
// Rebuild with simplate filesystem parsing thingy!

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

const (
    SIMPLATE_TMPL_{{.ConstName}} = __BACKTICK__{{.FirstTemplatePage.Body}}__BACKTICK__
)

var (
    simplateTmpl{{.FuncName}} = template.Must(template.New("{{.FuncName}}").Parse(SIMPLATE_TMPL_{{.ConstName}}))
)

func SimplateHandlerFunc{{.FuncName}}(w http.ResponseWriter, req *http.Request) {
    var err error
    ctx := make(map[string]interface{})
    response := goaspen.NewHTTPResponseWrapper(w, req)
    response.SetContentType("{{.ContentType}}")

    {{.LogicPage.Body}}

    var tmplBuf bytes.Buffer
    err = simplateTmpl{{.FuncName}}.Execute(&tmplBuf, ctx)
    if err != nil {
        response.Respond500(err)
        return
    }

    if err != nil {
        response.SetError(err)
    }
    response.SetBodyBytes(tmplBuf.Bytes())
    response.Respond()
}
`
	simplateTypeJSONTmpl = `
{{.InitPage.Body}}

func SimplateHandlerFunc{{.FuncName}}(w http.ResponseWriter, req *http.Request) {
    var err error
    ctx := make(map[string]interface{})
    response := goaspen.NewHTTPResponseWrapper(w, req)

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
	simplateTypeNegotiatedTmpl = `
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
)

func SimplateHandlerFunc{{.FuncName}}(w http.ResponseWriter, req *http.Request) {
    var err error
    ctx := make(map[string]interface{})
    response := goaspen.NewHTTPResponseWrapper(w, req)

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
)

func escapedSimplateTemplate(tmplString, name string) *template.Template {
	tmpl := template.New(name)
	tmplString = simplateTmplCommonHeader + tmplString
	escTmplString := strings.Replace(tmplString, "__BACKTICK__", "`", -1)
	return template.Must(tmpl.Parse(escTmplString))
}
