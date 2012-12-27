package goaspen_test

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"mime"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	. "github.com/meatballhat/goaspen"
)

const (
	BASIC_RENDERED_TXT_SIMPLATE = `
import (
    "time"
)

type RDance struct {
    Who  string
    When time.Time
}

ctx["D"] = &RDance{
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

type JDance struct {
    Who  string
    When time.Time
}

ctx["D"] = &JDance{
    Who:  "Everybody",
    When: time.Now(),
}
response.SetBody(ctx["D"])
`
	BASIC_NEGOTIATED_SIMPLATE = `
import (
    "time"
)

type NDance struct {
    Who  string
    When time.Time
}

ctx["D"] = &NDance{
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
		fmt.Sprintf("goaspen_test-%d", time.Now().UTC().UnixNano()))
	goAspenGenDir = path.Join(tmpdir, "src", "goaspen_gen")
	testSiteRoot  = path.Join(tmpdir, "test-site")
	goCmd         string
	noCleanup     bool
	testSiteFiles = map[string]string{
		"hams/bone/derp":                               BASIC_NEGOTIATED_SIMPLATE,
		"shill/cans.txt":                               BASIC_RENDERED_TXT_SIMPLATE,
		"hat/v.json":                                   BASIC_JSON_SIMPLATE,
		"silmarillion.handlebar.mustache.moniker.html": "<html>INVALID AS BUTT</html>",
		"Big CMS/Owns_UR Contents/flurb.txt":           BASIC_STATIC_TXT_SIMPLATE,
	}
)

func init() {
	err := os.Setenv("GOPATH", strings.Join([]string{tmpdir, os.Getenv("GOPATH")}, ":"))
	if err != nil {
		if noCleanup {
			log.Fatal(err)
		} else {
			panic(err)
		}
	}

	cmd, err := exec.LookPath("go")
	if err != nil {
		if noCleanup {
			log.Fatal(err)
		} else {
			panic(err)
		}
	}

	goCmdAddr := &goCmd
	*goCmdAddr = cmd

	noCleanupAddr := &noCleanup
	*noCleanupAddr = len(os.Getenv("GOASPEN_TEST_NOCLEANUP")) > 0
}

func mkTmpDir() {
	err := os.MkdirAll(goAspenGenDir, os.ModeDir|os.ModePerm)
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

func mkTestSite() string {
	mkTmpDir()

	for filePath, content := range testSiteFiles {
		fullPath := path.Join(testSiteRoot, filePath)
		err := os.MkdirAll(path.Dir(fullPath), os.ModeDir|os.ModePerm)
		if err != nil {
			panic(err)
		}

		f, err := os.Create(fullPath)
		if err != nil {
			panic(err)
		}

		_, err = f.WriteString(content)
		if err != nil {
			panic(err)
		}

		err = f.Close()
		if err != nil {
			panic(err)
		}
	}

	return testSiteRoot
}

func writeRenderedTemplate() (string, error) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		return "", err
	}

	outfileName := path.Join(goAspenGenDir, s.OutputName())
	outf, err := os.Create(outfileName)
	if err != nil {
		return outfileName, err
	}

	err = s.Execute(outf)
	if err != nil {
		return outfileName, err
	}

	err = outf.Close()
	if err != nil {
		return outfileName, err
	}

	return outfileName, nil
}

func runGoCommandOnGoAspenGen(command string) error {
	cmd := exec.Command(goCmd, command, "goaspen_gen")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return err
	}

	if noCleanup {
		fmt.Println(out.String())
	}
	return nil
}

func formatRenderedTemplate() error {
	return runGoCommandOnGoAspenGen("fmt")
}

func buildRenderedTemplate() error {
	return runGoCommandOnGoAspenGen("install")
}

func TestSimplateKnowsItsFilename(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/hasty-decisions.txt", "herpherpderpherp")
	if err != nil {
		t.Error(err)
		return
	}

	if s.Filename != "hasty-decisions.txt" {
		t.Errorf("Simplate filename incorrectly assigned as %s instead of %s",
			s.Filename, "hasty-decisions.txt")
	}
}

func TestSimplateKnowsItsContentType(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/hasty-decisions.js", "function herp() { return 'derp'; }")
	if err != nil {
		t.Error(err)
		return
	}

	expected := mime.TypeByExtension(".js")

	if s.ContentType != expected {
		t.Errorf("Simplate content type incorrectly assigned as %s instead of %s",
			s.ContentType, expected)
	}
}

func TestStaticSimplateKnowsItsOutputName(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/nothing.txt", "foo\nham\n")
	if err != nil {
		t.Error(err)
		return
	}

	if s.OutputName() != "nothing.txt" {
		t.Errorf("Static simplate output name is wrong!: %v", s.OutputName())
	}
}

func TestRenderedSimplateKnowsItsOutputName(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/flip/dippy slippy/snork.d/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.OutputName() != "flip-SLASH-dippy-SPACE-slippy-SLASH-snork-DOT-d-SLASH-basic-rendered-DOT-txt.go" {
		t.Errorf("Rendered simplate output name is wrong!: %v", s.OutputName())
	}
}

func TestDetectsRenderedSimplate(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SIMPLATE_TYPE_RENDERED {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_RENDERED)
	}
}

func TestDetectsStaticSimplate(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SIMPLATE_TYPE_STATIC {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_STATIC)
	}
}

func TestDetectsJSONSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SIMPLATE_TYPE_JSON {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_JSON)
	}
}

func TestDetectsNegotiatedSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/hork", BASIC_NEGOTIATED_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SIMPLATE_TYPE_NEGOTIATED {
		t.Errorf("Simplate detected as %s instead of %s",
			s.Type, SIMPLATE_TYPE_NEGOTIATED)
	}
}

func TestAssignsNoGoPagesToStaticSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.InitPage != nil {
		t.Errorf("Static simplate had init page assigned!: %v", s.InitPage)
	}

	if s.LogicPage != nil {
		t.Errorf("Static simplate had logic page assigned!: %v", s.LogicPage)
	}
}

func TestAssignsAnInitPageToRenderedSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.InitPage == nil {
		t.Errorf("Rendered simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsOneLogicPageToRenderedSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.LogicPage == nil {
		t.Errorf("Rendered simplate logic page not assigned!: %v", s.LogicPage)
	}
}

func TestAssignsOneTemplatePageToRenderedSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if len(s.TemplatePages) == 0 {
		t.Errorf("Rendered simplate had no template pages assigned!: %v", s.TemplatePages)
	}
}

func TestAssignsAnInitPageToJSONSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.InitPage == nil {
		t.Errorf("JSON simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsLogicPageToJSONSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.LogicPage == nil {
		t.Errorf("Rendered simplate logic page not assigned!: %v", s.LogicPage)
	}
}

func TestAssignsNoTemplatePageToJSONSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if len(s.TemplatePages) > 0 {
		t.Errorf("JSON simplate had template page(s) assigned!: %v", s.TemplatePages)
	}
}

func TestAssignsAnInitPageToNegotiatedSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-negotiated.txt", BASIC_NEGOTIATED_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.InitPage == nil {
		t.Errorf("Negotiated simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsALogicPageToNegotiatedSimplates(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-negotiated.txt", BASIC_NEGOTIATED_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.LogicPage == nil {
		t.Errorf("Negotiated simplate logic page not assigned!: %v", s.LogicPage)
	}
}

func TestRenderedSimplateCanExecuteToWriter(t *testing.T) {
	s, err := NewSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	var out bytes.Buffer
	err = s.Execute(&out)
	if err != nil {
		t.Error(err)
	}
}

func TestRenderedSimplateOutputIsValidGoSource(t *testing.T) {
	mkTmpDir()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	outfileName, err := writeRenderedTemplate()
	if err != nil {
		t.Error(err)
		return
	}

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, outfileName, nil, parser.DeclarationErrors)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestRenderedSimplateCanBeCompiled(t *testing.T) {
	mkTmpDir()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	_, err := writeRenderedTemplate()
	if err != nil {
		t.Error(err)
		return
	}

	err = formatRenderedTemplate()
	if err != nil {
		t.Error(err)
		return
	}

	err = buildRenderedTemplate()
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTreeWalkerRequiresValidDirectoryRoot(t *testing.T) {
	_, err := NewTreeWalker("/dev/null")
	if err == nil {
		t.Errorf("New tree walker failed to reject invalid dir!")
		return
	}
}

func TestTreeWalkerYieldsSimplates(t *testing.T) {
	siteRoot := mkTestSite()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	tw, err := NewTreeWalker(siteRoot)
	if err != nil {
		t.Error(err)
		return
	}

	n := 0

	simplates, err := tw.Simplates()
	if err != nil {
		t.Error(err)
	}

	for simplate := range simplates {
		if sort.SearchStrings(SIMPLATE_TYPES, simplate.Type) < 0 {
			t.Errorf("Simplate yielded with invalid type: %v", simplate.Type)
			return
		}
		n++
	}

	if n != 5 {
		t.Errorf("Tree walking yielded unexpected number of files: %v", n)
	}
}

func TestNewSiteBuilderRequiresValidRootDir(t *testing.T) {
	_, err := NewSiteBuilder("/dev/null", ".", false, false)
	if err == nil {
		t.Errorf("New site builder failed to reject invalid root dir!")
	}
}

func TestNewSiteBuilderRequiresValidOutputDir(t *testing.T) {
	_, err := NewSiteBuilder(".", "/dev/null", false, false)
	if err == nil {
		t.Errorf("New site builder failed to reject invalid output dir!")
	}
}

func TestNewSiteBuilderRequiresGofmtInPathIfFormatRequested(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	os.Setenv("PATH", "/bin")

	_, err := NewSiteBuilder(".", ".", true, false)
	if err == nil {
		t.Errorf("New site builder failed to reject based on missing 'gofmt'!")
	}
}

func TestSiteBuilderExposesRootDir(t *testing.T) {
	mkTestSite()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	sb, err := NewSiteBuilder(testSiteRoot, goAspenGenDir, true, true)
	if err != nil {
		t.Error(err)
		return
	}

	if sb.RootDir != testSiteRoot {
		t.Errorf("RootDir != %s: %s", testSiteRoot, sb.RootDir)
		return
	}
}

func TestSiteBuilderExposesOutputDir(t *testing.T) {
	mkTestSite()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	sb, err := NewSiteBuilder(testSiteRoot, goAspenGenDir, true, true)
	if err != nil {
		t.Error(err)
		return
	}

	if sb.OutputDir != goAspenGenDir {
		t.Errorf("OutputDir != %s: %s", goAspenGenDir, sb.OutputDir)
		return
	}
}

func TestSiteBuilderBuildWritesSources(t *testing.T) {
	mkTestSite()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	sb, err := NewSiteBuilder(testSiteRoot, goAspenGenDir, false, true)
	if err != nil {
		t.Error(err)
		return
	}

	sb.Build()

	fi, err := os.Stat(path.Join(goAspenGenDir, "shill-SLASH-cans-DOT-txt.go"))
	if err != nil {
		t.Error(err)
		return
	}

	if fi.Size() < int64(len(BASIC_RENDERED_TXT_SIMPLATE)) {
		t.Errorf("Generated file is too small! %v", fi.Size())
	}
}
