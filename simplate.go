package smplt

import (
	"strings"
)

const (
	SIMPLATE_TYPE_RENDERED   = "rendered"
	SIMPLATE_TYPE_STATIC     = "static"
	SIMPLATE_TYPE_NEGOTIATED = "negotiated"
)

type Simplate struct {
	Type string
}

func SimplateFromString(filename, content string) *Simplate {
	nbreaks := strings.Count(content, "")

	s := &Simplate{
		Type: SIMPLATE_TYPE_STATIC,
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
