package goaspen

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	InvalidTreeWalkerRoot = errors.New("Invalid tree walker root given")
)

type treeWalker struct {
	Root string
}

func NewTreeWalker(rootDir string) (*treeWalker, error) {
	fi, err := os.Stat(rootDir)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, InvalidTreeWalkerRoot
	}

	return &treeWalker{Root: rootDir}, nil
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

				smplt, err := NewSimplateFromString(me.Root, path, string(content))
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
