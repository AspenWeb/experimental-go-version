package aspen

import (
	"fmt"
	"net/http"
	"regexp"
)

const (
	internalAcceptHeader = "X-AspenGo-Accept"
	pathTransHeader      = "X-HTTP-Path-Translated"
)

var (
	vPathPart    = regexp.MustCompile("%([a-zA-Z_][-a-zA-Z0-9_]*)")
	vPathPartRep = "(?P<$1>[a-zA-Z_][-a-zA-Z0-9_]*)"
	nonAlNumDash = regexp.MustCompile("[^-a-zA-Z0-9]")
)

type handlerFuncRegistration struct {
	RequestPath string
	HandlerFunc http.HandlerFunc
	Negotiated  bool
	Virtual     bool
	Regexp      bool

	w *Website
}

func serve404(w http.ResponseWriter, req *http.Request) {
	charset := req.Header.Get("X-AspenGo-CharsetDynamic")
	if len(charset) == 0 {
		charset = "utf-8"
	}

	w.Header().Set("Content-Type", fmt.Sprintf("text/html; charset=%v", charset))
	w.WriteHeader(http.StatusNotFound)
	w.Write(http404Response)
}

// ripped right out of net/http/server.go, matches paths to longest similar
// path, which isn't exactly what we want.
func stdPathMatch(pattern, p string) bool {
	n := len(pattern)

	if n == 0 {
		return false
	}

	if pattern[n-1] != '/' {
		return pattern == p
	}

	return len(p) >= n && p[0:n] == pattern
}

func pathMatch(pattern, p string) bool {
	n := len(pattern)
	pathLen := len(p)

	if n == 0 {
		return false
	}

	if pattern[n-1] == '/' {
		pattern = pattern[:n-1]
	}

	if p[pathLen-1] == '/' {
		p = p[:pathLen-1]
	}

	return p == pattern
}
