package goaspen

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	DefaultGenPackage   = "goaspen_gen"
	DefaultOutputGopath = ""
	genServerTemplate   = template.Must(template.New("goaspen-genserver").Parse(`
package main
// GENERATED FILE - DO NOT EDIT
// Rebuild with goaspen-build!

import (
    "github.com/meatballhat/goaspen"
    _ "{{.GenPackage}}"
)

func main() {
    goaspen.RunServerMain("{{.WwwRoot}}",
        "{{.GenServerBind}}", "{{.GenPackage}}",
        "{{.CharsetDynamic}}", "{{.CharsetStatic}}",
        "{{.IndicesString}}", {{.ListDirs}}, {{.Debug}})
}
`))
)

type siteBuilder struct {
	// also used as defaults in generated server binary
	WwwRoot        string
	GenPackage     string
	GenServerBind  string
	CharsetStatic  string
	CharsetDynamic string
	Indices        []string
	ListDirs       bool
	Debug          bool

	// used primarily for compile time
	OutputGopath string
	Format       bool
	Compile      bool

	goexe       string
	walker      *treeWalker
	packagePath string
	genServer   string
	index       *siteIndex
}

type SiteBuilderCfg struct {
	WwwRoot       string
	OutputGopath  string
	GenPackage    string
	GenServerBind string
	Format        bool
	MkOutDir      bool
	Compile       bool

	CharsetStatic  string
	CharsetDynamic string
	Indices        []string
	ListDirs       bool
	Debug          bool
}

type siteIndex struct {
	WwwRoot   string                      `json:"root_dir"`
	Simplates map[string]*simplateSummary `json:"simplates"`
}

type simplateSummary struct {
	Type        string `json:"type"`
	ContentType string `json:"content_type"`
}

func init() {
	*(&DefaultOutputGopath) = strings.Split(os.Getenv("GOPATH"), ":")[0]
}

func newSiteBuilder(cfg *SiteBuilderCfg) (*siteBuilder, error) {
	var (
		err   error
		goexe string
	)

	rootDir, err := filepath.Abs(cfg.WwwRoot)
	if err != nil {
		return nil, err
	}

	outPath, err := filepath.Abs(cfg.OutputGopath)
	if err != nil {
		return nil, err
	}

	genPkg := cfg.GenPackage
	if len(genPkg) == 0 {
		genPkg = DefaultGenPackage
	}

	if cfg.Compile || cfg.Format {
		goexe, err = exec.LookPath("go")
		if err != nil {
			return nil, err
		}
	}

	if cfg.MkOutDir {
		err = os.MkdirAll(outPath, os.ModeDir|(os.FileMode)(0755))
		if err != nil {
			return nil, err
		}
	} else {
		outPathFi, err := os.Stat(outPath)
		if err != nil {
			return nil, err
		}

		if !outPathFi.IsDir() {
			return nil, fmt.Errorf("Invalid build output directory specified: %v", outPath)
		}
	}

	debugf("Creating new tree walker with package name %q, root dir %q", genPkg, rootDir)
	walker, err := newTreeWalker(genPkg, rootDir)
	if err != nil {
		return nil, err
	}

	sb := &siteBuilder{
		WwwRoot:       rootDir,
		OutputGopath:  outPath,
		GenPackage:    genPkg,
		GenServerBind: cfg.GenServerBind,
		Format:        cfg.Format,
		Compile:       cfg.Compile,

		CharsetDynamic: cfg.CharsetDynamic,
		CharsetStatic:  cfg.CharsetStatic,
		Indices:        cfg.Indices,
		ListDirs:       cfg.ListDirs,
		Debug:          cfg.Debug,

		goexe:       goexe,
		walker:      walker,
		packagePath: path.Join(outPath, "src", genPkg),
		genServer:   fmt.Sprintf("%s/%s-http-server", genPkg, genPkg),
		index: &siteIndex{
			WwwRoot:   rootDir,
			Simplates: map[string]*simplateSummary{},
		},
	}

	debugf("initialized site builder: %+v from cfg %+v", sb, cfg)

	return sb, nil
}

func (me *siteBuilder) writeOneSource(simplate *simplate) error {
	if simplate.Type == SimplateTypeStatic {
		return nil
	}

	outname := path.Join(me.packagePath, simplate.OutputName())
	debugf("Writing source for %v to %v\n", simplate.Filename, outname)

	outnameParent := path.Dir(outname)
	_, err := os.Stat(outnameParent)
	if err != nil {
		err = os.MkdirAll(outnameParent, os.ModeDir|(os.FileMode)(0755))
		if err != nil {
			return err
		}
	}

	outf, err := os.Create(outname)
	if err != nil {
		return err
	}

	debugf(" --> Executing simplate for %v\n", simplate.Filename)
	err = simplate.Execute(outf)
	if err != nil {
		return err
	}
	debugf(" --> Done executing simplate for %v\n", simplate.Filename)

	err = outf.Close()
	if err != nil {
		return err
	}

	debugf(" --> Returning nil after writing %v\n", outname)
	return nil
}

func (me *siteBuilder) writeGenServer() error {
	dirname := path.Join(me.OutputGopath, "src", me.genServer)
	err := os.MkdirAll(dirname, os.ModeDir|(os.FileMode)(0755))
	if err != nil {
		return err
	}

	fd, err := os.Create(path.Join(dirname, "main.go"))
	if err != nil {
		return err
	}

	defer fd.Close()

	err = genServerTemplate.Execute(fd, me)
	if err != nil {
		return err
	}

	return nil
}

func (me *siteBuilder) writeSources() error {
	simplates, err := me.walker.Simplates()
	if err != nil {
		return err
	}

	for simplate := range simplates {
		err := me.writeOneSource(simplate)
		if err != nil {
			return err
		}

		me.indexSimplate(simplate)
	}

	err = me.dumpSiteIndex()
	if err != nil {
		return err
	}

	err = me.writeGenServer()
	if err != nil {
		return err
	}

	return nil
}

func (me *siteBuilder) indexSimplate(simplate *simplate) {
	me.index.Simplates[fmt.Sprintf("/%v", simplate.Filename)] = &simplateSummary{
		Type:        simplate.Type,
		ContentType: simplate.ContentType,
	}
}

func (me *siteBuilder) dumpSiteIndex() error {
	idxPath := path.Join(me.WwwRoot, ".goaspen-index.json")

	out, err := os.Create(idxPath)
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(me.index, "", "  ")
	if err != nil {
		return err
	}

	_, err = out.Write(encoded)
	if err != nil {
		return err
	}

	err = out.Close()
	if err != nil {
		return err
	}

	return nil
}

func (me *siteBuilder) compileSources() error {
	origGopath := os.Getenv("GOPATH")
	err := os.Setenv("GOPATH", fmt.Sprintf("%s:%s", me.OutputGopath, origGopath))
	if err != nil {
		return err
	}

	defer os.Setenv("GOPATH", origGopath)

	installPkgCmd := exec.Command(me.goexe, "install", me.GenPackage)
	installPkgCmd.Stdout = os.Stdout
	installPkgCmd.Stderr = os.Stderr

	err = installPkgCmd.Run()
	if err != nil {
		return err
	}

	installBinCmd := exec.Command(me.goexe, "install", me.genServer)
	installBinCmd.Stdout = os.Stdout
	installPkgCmd.Stderr = os.Stderr

	err = installBinCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (me *siteBuilder) formatOneSource(sourceFile string) error {
	origGopath := os.Getenv("GOPATH")
	err := os.Setenv("GOPATH", me.OutputGopath)
	if err != nil {
		return err
	}

	defer os.Setenv("GOPATH", origGopath)

	formatCmd := exec.Command(me.goexe, "fmt", me.GenPackage)
	formatCmd.Stdout = os.Stdout
	formatCmd.Stderr = os.Stderr

	err = formatCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (me *siteBuilder) formatSources() error {
	sources, err := me.sourcesList()
	if err != nil {
		return err
	}

	for _, source := range sources {
		err = me.formatOneSource(source)
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *siteBuilder) sourcesList() ([]string, error) {
	return filepath.Glob(path.Join(me.packagePath, "*.go"))
}

func (me *siteBuilder) Build() error {
	err := me.writeSources()
	if err != nil {
		return err
	}

	if me.Format {
		err = me.formatSources()
		if err != nil {
			return err
		}
	}

	if me.Compile {
		err = me.compileSources()
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *siteBuilder) IndicesString() string {
	return strings.Join(me.Indices, ",")
}

/*
Build non-static simplates found in the given document root
(SiteBuilderCfg.WwwRoot) into Go sources written to the src dir of a given
GOPATH entry (SiteBuilderCfg.OutputGopath).  The generated package
(SiteBuilderCfg.GenPackage) will be used as the output source directory name
and written as the package declaration for each generated Go source file.  An
http server source will also be written to a directory nested within the
generated package.

Sources may be formatted via `gofmt` by setting the passed-in
SiteBuilderCfg.Format to true.  The generated package and http executable may
be automatically compiled by setting the passed-in SiteBuilderCfg.Compile to
true.

The generated server will support the following options, defaulted to the values
passed to BuildMain:

            --www_root, -w: Filesystem path of the document publishing root
     --network_address, -a: The IPv4 or IPv6 address to which the generated server
                            will bind by default
               --debug, -x: Print debugging output

*/
func BuildMain(cfg *SiteBuilderCfg) int {
	builder, err := newSiteBuilder(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	err = builder.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 2
	}

	return 0
}
