package goaspen

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	InvalidTreeWalkerRoot = errors.New("Invalid tree climber root given")
)

type TreeWalker struct {
	Root string
}

func NewTreeWalker(dirRoot string) (*TreeWalker, error) {
	fi, err := os.Stat(dirRoot)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, InvalidTreeWalkerRoot
	}

	return &TreeWalker{Root: dirRoot}, nil
}

func (me *TreeWalker) Simplates() (<-chan *Simplate, error) {
	schan := make(chan *Simplate)

	go func() {
		filepath.Walk(me.Root,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					log.Println("TreeWalker error:", err)
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

				schan <- NewSimplateFromString(path, string(content))
				return nil
			})
		close(schan)
	}()

	return (<-chan *Simplate)(schan), nil
}
