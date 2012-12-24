package smplt

import (
	"bufio"
	"io"
	"mime"
	"path"
	"strings"
)

const (
	SIMPLATE_TYPE_RENDERED   = "rendered"
	SIMPLATE_TYPE_STATIC     = "static"
	SIMPLATE_TYPE_NEGOTIATED = "negotiated"
	SIMPLATE_TYPE_JSON       = "json"
)

type Simplate struct {
	Filename     string
	Type         string
	ContentType  string
	InitPage     *SimplatePage
	LogicPages   []*SimplatePage
	TemplatePage *SimplatePage
}

type SimplatePage struct {
	Body string
}

func SimplateFromString(filename, content string) *Simplate {
	rawPages := strings.Split(content, "")
	nbreaks := len(rawPages) - 1

	s := &Simplate{
		Filename:    filename,
		Type:        SIMPLATE_TYPE_STATIC,
		ContentType: mime.TypeByExtension(path.Ext(filename)),
	}

	if nbreaks == 1 || nbreaks == 2 {
		s.InitPage = &SimplatePage{Body: rawPages[0]}
		s.LogicPages = append(s.LogicPages, &SimplatePage{Body: rawPages[1]})

		if s.ContentType == "application/json" {
			s.Type = SIMPLATE_TYPE_JSON
		} else {
			s.Type = SIMPLATE_TYPE_RENDERED
			s.TemplatePage = &SimplatePage{Body: rawPages[2]}
		}

		return s
	}

	if nbreaks > 2 {
		s.Type = SIMPLATE_TYPE_NEGOTIATED
		s.InitPage = &SimplatePage{Body: rawPages[0]}

		for _, rawPage := range rawPages {
			s.LogicPages = append(s.LogicPages, &SimplatePage{Body: rawPage})
		}

		return s
	}

	return s
}

func (me *Simplate) Execute(wr io.Writer, data interface{}) error {
	outbuf := bufio.NewWriter(wr)
	defer outbuf.Flush()

	_, err := outbuf.WriteString("package smplt_gen\n")
	if err != nil {
		return err
	}

	return nil
}
