package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/meatballhat/goaspen"
)

var (
	exampleUsage = `
Builds simplates found in ROOT_DIR (-d) into Go sources written to generated
package (-p) in the output GOPATH base (-o), optionally running 'go fmt' (-F).
The output GOPATH base must already exist, or the '-m' flag may be passed to
ensure it exists.

`
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", path.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, exampleUsage)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n")
}

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory! ", err)
	}

	outPath := goaspen.DefaultOutputGopath
	format := true
	mkOutDir := false
	debug := false
	compile := true
	genPkg := goaspen.DefaultGenPackage
	genServerBind := ":9182"

	flag.StringVar(&rootDir, "d", rootDir, "Root directory")
	flag.StringVar(&outPath, "o", outPath, "Output GOPATH base for generated sources")
	flag.StringVar(&genPkg, "p", genPkg, "Generated source package name")
	flag.StringVar(&genServerBind, "b", genServerBind, "Generated server binding")
	flag.BoolVar(&format, "F", format, "Format generated sources")
	flag.BoolVar(&mkOutDir, "m", mkOutDir, "Make output GOPATH base if not exists")
	flag.BoolVar(&debug, "x", debug, "Print debugging output")
	flag.BoolVar(&compile, "C", compile, "Compile generated sources")
	flag.Usage = usage
	flag.Parse()

	goaspen.SetDebug(debug)

	retcode := goaspen.BuildMain(&goaspen.SiteBuilderCfg{
		RootDir:       rootDir,
		OutputGopath:  outPath,
		GenPackage:    genPkg,
		GenServerBind: genServerBind,
		Format:        format,
		MkOutDir:      mkOutDir,
		Compile:       compile,
	})
	os.Exit(retcode)
}
