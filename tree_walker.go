package goaspen

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	InvalidTreeWalkerRoot = errors.New("Invalid tree walker root given")
)

type treeWalker struct {
	PackageName string
	Root        string
}

func newTreeWalker(packageName, rootDir string) (*treeWalker, error) {
	if len(packageName) == 0 {
		return nil, fmt.Errorf("Package name must be non-empty!")
	}

	fi, err := os.Stat(rootDir)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, InvalidTreeWalkerRoot
	}

	tw := &treeWalker{
		PackageName: packageName,
		Root:        rootDir,
	}

	return tw, nil
}

func (me *treeWalker) Simplates() (<-chan *simplate, error) {
	schan := make(chan *simplate)

	go func() {
		filepath.Walk(me.Root,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					debugf("TreeWalker error:", err)
					return nil
				}

				if info.IsDir() {
					return nil
				}

				f, err := os.Open(path)
				if err != nil {
					return err
				}

				defer f.Close()

				content, err := ioutil.ReadAll(f)
				if err != nil {
					return err
				}

				smplt, err := newSimplateFromString(me.PackageName,
					me.Root, path, string(content))
				if err != nil {
					return err
				}
				schan <- smplt
				return nil
			})
		close(schan)
	}()

	return (<-chan *simplate)(schan), nil
}
