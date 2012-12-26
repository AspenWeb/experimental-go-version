package goaspen

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type SiteBuilder struct {
	RootDir   string
	OutputDir string

	gofmt  string
	walker *TreeWalker
}

func NewSiteBuilder(rootDir, outDir string) (*SiteBuilder, error) {
	gofmt, err := exec.LookPath("gofmt")
	if err != nil {
		return nil, err
	}

	outDirFi, err := os.Stat(outDir)
	if err != nil {
		return nil, err
	}

	if !outDirFi.IsDir() {
		return nil, errors.New(fmt.Sprintf("Invalid build output directory specified: %v", outDir))
	}

	walker, err := NewTreeWalker(rootDir)
	if err != nil {
		return nil, err
	}

	sb := &SiteBuilder{
		RootDir:   rootDir,
		OutputDir: outDir,
		gofmt:     gofmt,
		walker:    walker,
	}

	return sb, nil
}

func (me *SiteBuilder) WriteSources() error {
	return nil
}

func (me *SiteBuilder) FormatSources() error {
	return nil
}

func BuildMain(rootDir, outDir string, format bool) int {
	builder, err := NewSiteBuilder(rootDir, outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	err = builder.WriteSources()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 2
	}

	if format {
		err = builder.FormatSources()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return 3
		}
	}

	return 0
}
