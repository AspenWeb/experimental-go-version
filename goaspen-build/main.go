package main

import (
	"flag"
	"log"
	"os"
	"path"

	"github.com/meatballhat/goaspen"
)

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory! ", err)
	}

	outDir := path.Join(rootDir, ".build", "src", "goaspen_gen")
	format := false

	flag.StringVar(&rootDir, "d", rootDir, "Root directory")
	flag.StringVar(&outDir, "o", outDir, "Output directory for generated sources")
	flag.BoolVar(&format, "f", format, "Format generated sources")
	flag.Parse()

	os.Exit(goaspen.BuildMain(rootDir, outDir, format))
}
