package smplt_test

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"mime"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/meatballhat/box-o-sand/gotime/smplt"
)

const (
	BASIC_RENDERED_TXT_SIMPLATE = `
import (
    "time"
)

type Dance struct {
    Who  string
    When time.Time
}

D := &Dance{
    Who:  "Everybody",
    When: time.Now(),
}

{{.D.Who}} Dance {{.D.When}}!
`
	BASIC_STATIC_TXT_SIMPLATE = `
Everybody Dance Now!
`
	BASIC_JSON_SIMPLATE = `
import (
    "time"
)

type Dance struct {
    Who  string
    When time.Time
}

D := &Dance{
    Who:  "Everybody",
    When: time.Now(),
}
response.SetBody(D)
`
	BASIC_NEGOTIATED_SIMPLATE = `
import (
    "time"
)

type Dance struct {
    Who  string
    When time.Time
}

D := &Dance{
    Who:  "Everybody",
    When: time.Now(),
}
 text/plain
{{.D.Who}} Dance {{.D.When}}!

 application/json
{"who":"{{.D.Who}}","when":"{{.D.When}}"}
`
)

var (
	tmpdir = path.Join(os.TempDir(),
		fmt.Sprintf("smplt_test-%d", time.Now().UTC().UnixNano()))
)

func mkTmpDir() {
	err := os.MkdirAll(tmpdir, os.ModeDir|os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func rmTmpDir() {
	err := os.RemoveAll(tmpdir)
	if err != nil {
		panic(err)
	}
}

func TestSimplateKnowsItsFilename(t *testing.T) {
	s := SimplateFromString("hasty-decisions.txt", "herpherpderpherp")
	if s.Filename != "hasty-decisions.txt" {
		t.Errorf("Simplate filename incorrectly assigned as %s instead of %s",
			s.Filename, "hasty-decisions.txt")
	}
}

func TestSimplateKnowsItsContentType(t *testing.T) {
	s := SimplateFromString("hasty-decisions.js", "function herp() { return 'derp'; }")
	expected := mime.TypeByExtension(".js")

	if s.ContentType != expected {
		t.Errorf("Simplate content type incorrectly assigned as %s instead of %s",
			s.ContentType, expected)
	}
}

func TestStaticSimplateKnowsItsOutputName(t *testing.T) {
	s := SimplateFromString("nothing.txt", "foo\nham\n")
	if s.OutputName() != "nothing.txt" {
		t.Errorf("Static simplate output name is wrong!: %v", s.OutputName())
	}
}

func TestRenderedSimplateKnowsItsOutputName(t *testing.T) {
	s := SimplateFromString("flip/dippy slippy/snork.d/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if s.OutputName() != "flip-SLASH-dippy-SPACE-slippy-SLASH-snork-DOT-d-SLASH-basic-rendered-DOT-txt.go" {
		t.Errorf("Rendered simplate output name is wrong!: %v", s.OutputName())
	}
}

func TestDetectsRenderedSimplate(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_RENDERED {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_RENDERED)
	}
}

func TestDetectsStaticSimplate(t *testing.T) {
	s := SimplateFromString("basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_STATIC {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_STATIC)
	}
}

func TestDetectsJSONSimplates(t *testing.T) {
	s := SimplateFromString("basic.json", BASIC_JSON_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_JSON {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_JSON)
	}
}

func TestDetectsNegotiatedSimplates(t *testing.T) {
	s := SimplateFromString("hork", BASIC_NEGOTIATED_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_NEGOTIATED {
		t.Errorf("Simplate detected as %s instead of %s",
			s.Type, SIMPLATE_TYPE_NEGOTIATED)
	}
}

func TestAssignsNoGoPagesToStaticSimplates(t *testing.T) {
	s := SimplateFromString("basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
	if s.InitPage != nil {
		t.Errorf("Static simplate had init page assigned!: %v", s.InitPage)
	}

	if len(s.LogicPages) > 0 {
		t.Errorf("Static simplate had logic pages assigned!: %v", s.LogicPages)
	}
}

func TestAssignsAnInitPageToRenderedSimplates(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if s.InitPage == nil {
		t.Errorf("Rendered simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsOneLogicPageToRenderedSimplates(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if len(s.LogicPages) != 1 {
		t.Errorf("Rendered simplate unexpected number "+
			"of logic pages assigned!: %v", len(s.LogicPages))
	}
}

func TestAssignsOneTemplatePageToRenderedSimplates(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if s.TemplatePage == nil {
		t.Errorf("Rendered simplate had no template page assigned!: %v", s.TemplatePage)
	}
}

func TestAssignsAnInitPageToJSONSimplates(t *testing.T) {
	s := SimplateFromString("basic.json", BASIC_JSON_SIMPLATE)
	if s.InitPage == nil {
		t.Errorf("JSON simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsOneLogicPageToJSONSimplates(t *testing.T) {
	s := SimplateFromString("basic.json", BASIC_JSON_SIMPLATE)
	if len(s.LogicPages) != 1 {
		t.Errorf("Rendered simplate unexpected number "+
			"of logic pages assigned!: %v", len(s.LogicPages))
	}
}

func TestAssignsNoTemplatePageToJSONSimplates(t *testing.T) {
	s := SimplateFromString("basic.json", BASIC_JSON_SIMPLATE)
	if s.TemplatePage != nil {
		t.Errorf("JSON simplate had a template page assigned!: %v", s.TemplatePage)
	}
}

func TestAssignsAnInitPageToNegotiatedSimplates(t *testing.T) {
	s := SimplateFromString("basic-negotiated.txt", BASIC_NEGOTIATED_SIMPLATE)
	if s.InitPage == nil {
		t.Errorf("Negotiated simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsAtLeastOneLogicPageToNegotiatedSimplates(t *testing.T) {
	s := SimplateFromString("basic-negotiated.txt", BASIC_NEGOTIATED_SIMPLATE)
	if len(s.LogicPages) < 1 {
		t.Errorf("Negotiated simplate unexpected number "+
			"of logic pages assigned!: %v", len(s.LogicPages))
	}
}

func TestAssignsNoTemplatePageToNegotiatedSimplates(t *testing.T) {
	s := SimplateFromString("basic-negotiated.txt", BASIC_NEGOTIATED_SIMPLATE)
	if s.TemplatePage != nil {
		t.Errorf("Negotiated simplate had a template page assigned!: %v", s.TemplatePage)
	}
}

func TestRenderedSimplateCanExecuteToWriter(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	var out bytes.Buffer
	err := s.Execute(&out)
	if err != nil {
		t.Error(err)
	}
}

func TestRenderedSimplateOutputIsValidGoSource(t *testing.T) {
	mkTmpDir()
	defer rmTmpDir()

	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	outfile_name := path.Join(tmpdir, s.OutputName())
	outf, err := os.Create(outfile_name)
	if err != nil {
		t.Error(err)
		return
	}

	s.Execute(outf)
	err = outf.Close()
	if err != nil {
		t.Error(err)
	}

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, outfile_name, nil, parser.DeclarationErrors)
	if err != nil {
		t.Error(err)
		return
	}
}
