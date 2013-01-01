package goaspen

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

const (
	internalAcceptHeader = "X-GoAspen-Accept"
)

type handlerFuncRegistration struct {
	RequestPath string
	HandlerFunc func(http.ResponseWriter, *http.Request)
	Receiver    *directoryHandler

	app *App
}

type directoryHandler struct {
	WwwRoot         string
	DirectoryPath   string
	PatternHandlers map[string]*handlerFuncRegistration

	app *App
}

func (me *directoryHandler) Handle(w http.ResponseWriter, req *http.Request) {
	debugf("Handling directory response for %q", req.URL.Path)
	me.updateNegType(req, req.URL.Path)

	for requestUri, handler := range me.PatternHandlers {
		matched, err := path.Match(requestUri, req.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(http500Response)
		}

		if matched {
			defer handler.HandlerFunc(w, req)
			return
		}

	}

	err := me.serveStatic(w, req)
	if err != nil {
		if strings.HasSuffix(req.URL.Path, "/favicon.ico") {
			w.Header().Set("Content-Type", "image/x-icon")
			w.WriteHeader(http.StatusOK)
			w.Write(faviconIco)
			return
		}

		w.Header().Set("Content-Type",
			fmt.Sprintf("text/html; charset=%v", me.app.CharsetDynamic))
		w.WriteHeader(http.StatusNotFound)
		w.Write(http404Response)
	}
}

func (me *directoryHandler) AddGlob(pathGlob string,
	reg *handlerFuncRegistration) error {

	debugf("Adding glob %q to pattern handlers for %q", pathGlob, me.DirectoryPath)
	me.PatternHandlers[pathGlob] = reg
	return nil
}

func (me *directoryHandler) serveStatic(w http.ResponseWriter, req *http.Request) error {
	fullPath, err := me.findStaticPath(req)
	if err != nil {
		return err
	}

	debugf("Found static path path %q from root %q and request path %q",
		fullPath, me.WwwRoot, req.URL.Path)

	fi, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return fmt.Errorf("Cannot serve directory at %q", fullPath)
	}

	ctype := mime.TypeByExtension(path.Ext(fullPath))
	if strings.HasPrefix(ctype, "text/") && !strings.Contains(ctype, "charset=") {
		ctype = fmt.Sprintf("%v; charset=utf-8", ctype)
	}

	outf, err := os.Open(fullPath)
	if err != nil {
		debugf("Could not open %q", fullPath)
		return err
	}

	defer outf.Close()

	w.Header().Set("Content-Length", fmt.Sprintf("%v", fi.Size()))
	w.Header().Set("Content-Type", ctype)
	w.WriteHeader(http.StatusOK)
	io.Copy(w, outf)

	return nil
}

func (me *directoryHandler) findStaticPath(req *http.Request) (string, error) {
	fullPath := path.Join(me.WwwRoot, strings.TrimLeft(req.URL.Path, "/"))

	fi, err := os.Stat(fullPath)
	if _, ok := err.(*os.PathError); ok || fi.IsDir() {
		debugf("Either could not stat or is dir %q", fullPath)
		debugf("Looking for candidate index files.  Configured indices = %+v",
			me.app.Indices)

		for _, idx := range me.app.Indices {
			if len(idx) == 0 {
				continue
			}

			tryFullPath := path.Join(fullPath, idx)

			debugf("Checking for candidate index file at %q", tryFullPath)
			fi, err := os.Stat(tryFullPath)
			if err != nil || fi.IsDir() {
				continue
			}

			debugf("Found candidate index file at %q", tryFullPath)
			return tryFullPath, nil
		}

		return "", err
	}

	return fullPath, nil
}

func (me *directoryHandler) updateNegType(req *http.Request, filename string) {
	mediaType := mime.TypeByExtension(path.Ext(filename))
	if len(mediaType) == 0 {
		mediaType = "text/html" // FIXME get default from config
	}

	req.Header.Set(internalAcceptHeader, mediaType)
}
