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
	simplateTmplWebFuncDeclaration = `
    local{{.FuncName}}Website = goaspen.DeclareWebsite("{{.GenPackage}}")
    _ = local{{.FuncName}}Website.NewHandlerFuncRegistration("/{{.Filename}}", SimplateHandlerFunc{{.FuncName}})
`
	simplateTmplFuncHeader = `
func SimplateHandlerFunc{{.FuncName}}(w http.ResponseWriter, request *http.Request) {
    var err error
    website := local{{.FuncName}}Website

    response := website.NewHTTPResponseWrapper(w, request)

    __file__ := "{{.AbsFilename}}"
    ctx := make(map[string]interface{})

    {{.LogicPage.Body}}
`
	simplateTmplFuncFooter = `
    response.NegotiateAndCallHandler()
    if err != nil {
        response.SetError(err)
    }

    response.DebugContext(__file__, ctx)
`

	simplateTypeRenderedTmpl = simplateTmplCommonHeader + `
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
    ` + simplateTmplWebFuncDeclaration + `
)

` + simplateTmplFuncHeader + `

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

` + simplateTmplFuncFooter + `
    response.Respond()
}
`
	simplateTypeJSONTmpl = simplateTmplCommonHeader + `
{{.InitPage.Body}}

var (
` + simplateTmplWebFuncDeclaration + `
)

` + simplateTmplFuncHeader + simplateTmplFuncFooter + `
    response.RespondJSON()
}
`
	simplateTypeNegotiatedTmpl = simplateTypeRenderedTmpl
)

func escapedSimplateTemplate(tmplString, name string) *template.Template {
	tmpl := template.New(name)
	escTmplString := strings.Replace(tmplString, "__BACKTICK__", "`", -1)
	return template.Must(tmpl.Parse(escTmplString))
}
