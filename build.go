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
	Format    bool

	gofmt  string
	walker *TreeWalker
}

func NewSiteBuilder(rootDir, outDir string, format, mkOutDir bool) (*SiteBuilder, error) {
	var (
		err   error
		gofmt string
	)

	if format {
		gofmt, err = exec.LookPath("gofmt")
		if err != nil {
			return nil, err
		}
	}

	if mkOutDir {
		err = os.MkdirAll(outDir, os.ModeDir|os.ModePerm)
		if err != nil {
			return nil, err
		}
	} else {
		outDirFi, err := os.Stat(outDir)
		if err != nil {
			return nil, err
		}

		if !outDirFi.IsDir() {
			return nil, errors.New(fmt.Sprintf("Invalid build output directory specified: %v", outDir))
		}
	}

	walker, err := NewTreeWalker(rootDir)
	if err != nil {
		return nil, err
	}

	sb := &SiteBuilder{
		RootDir:   rootDir,
		OutputDir: outDir,
		Format:    format,
		gofmt:     gofmt,
		walker:    walker,
	}

	return sb, nil
}

func (me *SiteBuilder) writeSources() error {
	return nil
}

func (me *SiteBuilder) formatSources() error {
	return nil
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

	return nil
}

func BuildMain(rootDir, outDir string, format, mkOutDir bool) int {
	builder, err := NewSiteBuilder(rootDir, outDir, format, mkOutDir)
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
