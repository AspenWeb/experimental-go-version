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
}

func SimplateFromString(filename, content string) *Simplate {
	nbreaks := strings.Count(content, "")

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
		return s
	}

	if nbreaks > 2 {
		s.Type = SIMPLATE_TYPE_NEGOTIATED
		return s
	}

	return s
}
