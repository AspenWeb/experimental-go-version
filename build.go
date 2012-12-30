package goaspen

import (
	"bytes"
	"errors"
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
	DefaultOutputGopath string
	defaultOutDirAddr   = &DefaultOutputGopath
	genServerTemplate   = template.Must(template.New("goaspen-genserver").Parse(`
package main
// GENERATED FILE - DO NOT EDIT
// Rebuild with goaspen-build!

import (
    "flag"
    "log"

    "github.com/meatballhat/goaspen"
    _ "{{.GenPackage}}"
)

var (
    siteRoot = flag.String("d", "{{.RootDir}}", "Site root directory (for serving static files.)")
    serverBind = flag.String("b", "{{.GenServerBind}}", "Server binding")
    debug = flag.Bool("x", false, "Print debug output")
)

func main() {
    flag.Parse()
    goaspen.SetDebug(*debug)

    err := goaspen.RunServer("{{.GenPackage}}", *serverBind, *siteRoot)
    if err != nil {
        log.Fatal(err)
    }
}
`))
)

type SiteBuilder struct {
	RootDir       string
	OutputGopath  string
	GenPackage    string
	GenServerBind string
	Format        bool
	Compile       bool

	goexe       string
	walker      *TreeWalker
	packagePath string
	genServer   string
}

type SiteBuilderCfg struct {
	RootDir       string
	OutputGopath  string
	GenPackage    string
	GenServerBind string
	Format        bool
	MkOutDir      bool
	Compile       bool
}

func init() {
	*defaultOutDirAddr = strings.Split(os.Getenv("GOPATH"), ":")[0]
}

func NewSiteBuilder(cfg *SiteBuilderCfg) (*SiteBuilder, error) {
	var (
		err   error
		goexe string
	)

	rootDir, err := filepath.Abs(cfg.RootDir)
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
			return nil, errors.New(fmt.Sprintf("Invalid build output directory specified: %v", outPath))
		}
	}

	walker, err := NewTreeWalker(rootDir)
	if err != nil {
		return nil, err
	}

	sb := &SiteBuilder{
		RootDir:       rootDir,
		OutputGopath:  outPath,
		GenPackage:    genPkg,
		GenServerBind: cfg.GenServerBind,
		Format:        cfg.Format,
		Compile:       cfg.Compile,

		goexe:       goexe,
		walker:      walker,
		packagePath: path.Join(outPath, "src", genPkg),
		genServer:   fmt.Sprintf("%s/%s-http-server", genPkg, genPkg),
	}

	return sb, nil
}

func (me *SiteBuilder) writeOneSource(simplate *Simplate) error {
	if simplate.Type == SIMPLATE_TYPE_STATIC {
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

func (me *SiteBuilder) writeGenServer() error {
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

func (me *SiteBuilder) writeSources() error {
	simplates, err := me.walker.Simplates()
	if err != nil {
		return err
	}

	for simplate := range simplates {
		err := me.writeOneSource(simplate)
		if err != nil {
			return err
		}
	}

	err = me.writeGenServer()
	if err != nil {
		return err
	}

	return nil
}

func (me *SiteBuilder) compileSources() error {
	origGopath := os.Getenv("GOPATH")
	err := os.Setenv("GOPATH", fmt.Sprintf("%s:%s", me.OutputGopath, origGopath))
	if err != nil {
		return err
	}

	defer os.Setenv("GOPATH", origGopath)

	var out bytes.Buffer
	installPkgCmd := exec.Command(me.goexe, "install", me.GenPackage)
	installPkgCmd.Stdout = &out

	err = installPkgCmd.Run()
	if err != nil {
		return err
	}

	installBinCmd := exec.Command(me.goexe, "install", me.genServer)
	//installBinCmd.Stdout = &out

	err = installBinCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (me *SiteBuilder) formatOneSource(sourceFile string) error {
	origGopath := os.Getenv("GOPATH")
	err := os.Setenv("GOPATH", me.OutputGopath)
	if err != nil {
		return err
	}

	defer os.Setenv("GOPATH", origGopath)

	var out bytes.Buffer
	formatCmd := exec.Command(me.goexe, "fmt", me.GenPackage)
	formatCmd.Stdout = &out

	err = formatCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (me *SiteBuilder) formatSources() error {
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

func (me *SiteBuilder) sourcesList() ([]string, error) {
	return filepath.Glob(path.Join(me.packagePath, "*.go"))
}

func (me *SiteBuilder) Build() error {
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

func BuildMain(cfg *SiteBuilderCfg) int {
	builder, err := NewSiteBuilder(cfg)
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
