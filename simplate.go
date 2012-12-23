package smplt

import (
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
	Filename    string
	Type        string
	ContentType string
	InitPage    *SimplatePage
	LogicPages  []*SimplatePage
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

	if nbreaks == 1 && s.ContentType == "application/json" {
		s.Type = SIMPLATE_TYPE_JSON
		return s
	}

	if nbreaks == 2 {
		s.Type = SIMPLATE_TYPE_RENDERED
		s.InitPage = &SimplatePage{Body: rawPages[0]}
		s.LogicPages = append(s.LogicPages, &SimplatePage{Body: rawPages[1]})
		return s
	}

	if nbreaks > 2 {
		s.Type = SIMPLATE_TYPE_NEGOTIATED
		return s
	}

	return s
}
