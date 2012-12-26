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
Builds simplates found in ROOT_DIR (-d) into Go sources
written to OUTPUT_DIR (-o), optionally passing them through
the 'gofmt' tool (-f).  The output dir must already exist, or
the '-m' flag may be passed to ensure it exists.

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

	outDir := path.Join(rootDir, ".build", "src", "goaspen_gen")
	format := false
	mkOutDir := false

	flag.StringVar(&rootDir, "d", rootDir, "Root directory")
	flag.StringVar(&outDir, "o", outDir, "Output directory for generated sources")
	flag.BoolVar(&format, "f", format, "Format generated sources")
	flag.BoolVar(&mkOutDir, "m", mkOutDir, "Make output directory if not exists")
	flag.Usage = usage
	flag.Parse()

	retcode := goaspen.BuildMain(rootDir, outDir, format, mkOutDir)
	os.Exit(retcode)
}
