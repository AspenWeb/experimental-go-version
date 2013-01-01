package goaspen

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"testing"
	"time"
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
	testWwwRoot   = path.Join(tmpdir, "test-site")
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
		fullPath := path.Join(testWwwRoot, filePath)
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

	return testWwwRoot
}

func writeRenderedTemplate() (string, error) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
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

func formatGoAspenGen() error {
	return runGoCommandOnGoAspenGen("fmt")
}

func buildGoAspenGen() error {
	return runGoCommandOnGoAspenGen("install")
}

func TestSimplateKnowsItsFilename(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/hasty-decisions.txt", "herpherpderpherp")
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
	s, err := newSimplateFromString("/tmp", "/tmp/hasty-decisions.js", "function herp() { return 'derp'; }")
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
	s, err := newSimplateFromString("/tmp", "/tmp/nothing.txt", "foo\nham\n")
	if err != nil {
		t.Error(err)
		return
	}

	if s.OutputName() != "nothing.txt" {
		t.Errorf("Static simplate output name is wrong!: %v", s.OutputName())
	}
}

func TestRenderedSimplateKnowsItsOutputName(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/flip/dippy slippy/snork.d/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.OutputName() != "flip-SLASH-dippy-SPACE-slippy-SLASH-snork-DOT-d-SLASH-basic-rendered-DOT-txt.go" {
		t.Errorf("Rendered simplate output name is wrong!: %v", s.OutputName())
	}
}

func TestDetectsRenderedSimplate(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SimplateTypeRendered {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SimplateTypeRendered)
	}
}

func TestDetectsStaticSimplate(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SimplateTypeStatic {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SimplateTypeStatic)
	}
}

func TestDetectsJSONSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SimplateTypeJson {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SimplateTypeJson)
	}
}

func TestDetectsNegotiatedSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/hork", BASIC_NEGOTIATED_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.Type != SimplateTypeNegotiated {
		t.Errorf("Simplate detected as %s instead of %s",
			s.Type, SimplateTypeNegotiated)
	}
}

func TestAssignsNoGoPagesToStaticSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
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
	s, err := newSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.InitPage == nil {
		t.Errorf("Rendered simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsOneLogicPageToRenderedSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.LogicPage == nil {
		t.Errorf("Rendered simplate logic page not assigned!: %v", s.LogicPage)
	}
}

func TestAssignsOneTemplatePageToRenderedSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if len(s.TemplatePages) == 0 {
		t.Errorf("Rendered simplate had no template pages assigned!: %v", s.TemplatePages)
	}
}

func TestAssignsAnInitPageToJSONSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.InitPage == nil {
		t.Errorf("JSON simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsLogicPageToJSONSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.LogicPage == nil {
		t.Errorf("Rendered simplate logic page not assigned!: %v", s.LogicPage)
	}
}

func TestAssignsNoTemplatePageToJSONSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic.json", BASIC_JSON_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if len(s.TemplatePages) > 0 {
		t.Errorf("JSON simplate had template page(s) assigned!: %v", s.TemplatePages)
	}
}

func TestAssignsAnInitPageToNegotiatedSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-negotiated.txt", BASIC_NEGOTIATED_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.InitPage == nil {
		t.Errorf("Negotiated simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsALogicPageToNegotiatedSimplates(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-negotiated.txt", BASIC_NEGOTIATED_SIMPLATE)
	if err != nil {
		t.Error(err)
		return
	}

	if s.LogicPage == nil {
		t.Errorf("Negotiated simplate logic page not assigned!: %v", s.LogicPage)
	}
}

func TestRenderedSimplateCanExecuteToWriter(t *testing.T) {
	s, err := newSimplateFromString("/tmp", "/tmp/basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
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

	err = formatGoAspenGen()
	if err != nil {
		t.Error(err)
		return
	}

	err = buildGoAspenGen()
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTreeWalkerRequiresValidDirectoryRoot(t *testing.T) {
	_, err := newTreeWalker("/dev/null")
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

	tw, err := newTreeWalker(siteRoot)
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
		if sort.SearchStrings(SimplateTypes, simplate.Type) < 0 {
			t.Errorf("Simplate yielded with invalid type: %v", simplate.Type)
			return
		}
		n++
	}

	if n != 5 {
		t.Errorf("Tree walking yielded unexpected number of files: %v", n)
	}
}

func TestNewSiteBuilderRequiresValidWwwRoot(t *testing.T) {
	_, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       "/dev/null",
		OutputGopath:  ".",
		GenServerBind: ":9182",
	})
	if err == nil {
		t.Errorf("New site builder failed to reject invalid root dir!")
	}
}

func TestNewSiteBuilderRequiresValidOutputDir(t *testing.T) {
	_, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       ".",
		OutputGopath:  "/dev/null",
		GenServerBind: ":9182",
	})
	if err == nil {
		t.Errorf("New site builder failed to reject invalid output dir!")
	}
}

func TestNewSiteBuilderDefaultsGeneratedCodePackage(t *testing.T) {
	sb, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       ".",
		OutputGopath:  ".",
		GenServerBind: ":9182",
	})

	if err != nil {
		t.Error(err)
	}

	if sb.GenPackage != "goaspen_gen" {
		t.Errorf("Generated package default != \"goaspen_gen\": %q", sb.GenPackage)
	}
}

func TestSiteBuilderExposesWwwRoot(t *testing.T) {
	mkTestSite()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	sb, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       testWwwRoot,
		OutputGopath:  tmpdir,
		GenServerBind: ":9182",
		Format:        true,
		MkOutDir:      true,
	})
	if err != nil {
		t.Error(err)
		return
	}

	if sb.WwwRoot != testWwwRoot {
		t.Errorf("WwwRoot != %s: %s", testWwwRoot, sb.WwwRoot)
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

	sb, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       testWwwRoot,
		OutputGopath:  tmpdir,
		GenServerBind: ":9182",
		Format:        true,
		MkOutDir:      true,
	})
	if err != nil {
		t.Error(err)
		return
	}

	if sb.OutputGopath != tmpdir {
		t.Errorf("OutputDir != %s: %s", tmpdir, sb.OutputGopath)
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

	sb, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       testWwwRoot,
		OutputGopath:  tmpdir,
		GenServerBind: ":9182",
		MkOutDir:      true,
		Compile:       false,
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = sb.Build()
	if err != nil {
		t.Error(err)
		return
	}

	fi, err := os.Stat(path.Join(goAspenGenDir, "shill-SLASH-cans-DOT-txt.go"))
	if err != nil {
		t.Error(err)
		return
	}

	if fi.Size() < int64(len(BASIC_RENDERED_TXT_SIMPLATE)) {
		t.Errorf("Generated file is too small! %v", fi.Size())
	}
}

func TestSiteBuilderBuildFormatsSources(t *testing.T) {
	mkTestSite()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	sb, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       testWwwRoot,
		OutputGopath:  tmpdir,
		GenServerBind: ":9182",
		Format:        true,
		MkOutDir:      true,
		Compile:       false,
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = sb.Build()
	if err != nil {
		t.Error(err)
		return
	}

	fileName := path.Join(goAspenGenDir, "shill-SLASH-cans-DOT-txt.go")

	fd, err := os.Open(fileName)
	if err != nil {
		t.Error(err)
		return
	}

	fileContent, err := ioutil.ReadAll(fd)
	if err != nil {
		t.Error(err)
		return
	}

	err = fd.Close()
	if err != nil {
		t.Error(err)
		return
	}

	firstHash := sha1.New()
	io.WriteString(firstHash, string(fileContent))
	firstSum := fmt.Sprintf("%x", firstHash.Sum(nil))

	err = formatGoAspenGen()
	if err != nil {
		t.Error(err)
		return
	}

	fd, err = os.Open(fileName)
	if err != nil {
		t.Error(err)
		return
	}

	fileContent, err = ioutil.ReadAll(fd)
	if err != nil {
		t.Error(err)
		return
	}

	secondHash := sha1.New()
	io.WriteString(secondHash, string(fileContent))
	secondSum := fmt.Sprintf("%x", secondHash.Sum(nil))

	if firstSum != secondSum {
		t.Errorf("Hash for %q changed!", fileName)
	}
}

func TestNewSiteBuilderCompilesSources(t *testing.T) {
	mkTestSite()
	if noCleanup {
		fmt.Println("tmpdir =", tmpdir)
	} else {
		defer rmTmpDir()
	}

	sb, err := newSiteBuilder(&SiteBuilderCfg{
		WwwRoot:       testWwwRoot,
		OutputGopath:  tmpdir,
		GenServerBind: ":9182",
		Format:        true,
		Compile:       true,
		MkOutDir:      true,
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = sb.Build()
	if err != nil {
		t.Error(err)
		return
	}

	serverBinary := path.Join(sb.OutputGopath, "bin", "goaspen_gen-http-server")

	fi, err := os.Stat(serverBinary)
	if err != nil {
		t.Error(err)
		return
	}

	if fi.Mode() != (os.FileMode)(0750) {
		t.Errorf("Site server binary %q permissions != %v: %v",
			serverBinary, (os.FileMode)(0750), fi.Mode())
	}
}
