package goaspen

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
)

const (
	internalAcceptHeader = "X-GoAspen-Accept"
)

var (
	handlerFuncRegistrations = map[string]*handlerFuncRegistration{}

	regLock sync.RWMutex
)

type handlerFuncRegistration struct {
	RequestPath string
	HandlerFunc func(http.ResponseWriter, *http.Request)
	Receiver    *directoryHandler
}

type directoryHandler struct {
	SiteRoot        string
	DirectoryPath   string
	PatternHandlers map[string]*handlerFuncRegistration
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

	err := me.ServeStatic(w, req)
	if err != nil {
		if strings.HasSuffix(req.URL.Path, "/favicon.ico") {
			w.Header().Set("Content-Type", "image/x-icon")
			w.WriteHeader(http.StatusOK)
			w.Write(faviconIco)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
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

func (me *directoryHandler) ServeStatic(w http.ResponseWriter, req *http.Request) error {
	return errors.New("Not implemented, so pretending nothing is here!")
}

func (me *directoryHandler) updateNegType(req *http.Request, filename string) {
	mediaType := mime.TypeByExtension(path.Ext(filename))
	if len(mediaType) == 0 {
		mediaType = "text/html" // FIXME get default from config
	}

	req.Header.Set(internalAcceptHeader, mediaType)
}

func NewHandlerFuncRegistration(requestPath string,
	handler func(http.ResponseWriter, *http.Request)) *handlerFuncRegistration {

	if len(requestPath) < 1 {
		panic(fmt.Errorf("Invalid request path %q", requestPath))
	}

	regLock.RLock()
	defer regLock.RUnlock()

	handlerFuncRegistrations[requestPath] = &handlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
	}

	if !strings.HasSuffix(requestPath, "/") && len(path.Ext(requestPath)) == 0 {
		pathGlob := requestPath + ".*"
		handlerFuncRegistrations[pathGlob] = &handlerFuncRegistration{
			RequestPath: pathGlob,
			HandlerFunc: handler,
		}
	}

	return handlerFuncRegistrations[requestPath]
}

func expandAllHandlerFuncRegistrations() error {
	debugf("Expanding all handler func registrations!")

	for _, reg := range handlerFuncRegistrations {
		err := expandHandlerFuncRegistration(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func expandHandlerFuncRegistration(reg *handlerFuncRegistration) error {
	if path.Ext(reg.RequestPath) == ".*" {
		debugf("Found glob registration %q, adding to directory handler",
			reg.RequestPath)

		err := addGlobToDirectoryHandler(path.Dir(reg.RequestPath),
			reg.RequestPath, reg.HandlerFunc)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func registerAllHandlerFuncs() error {
	debugf("Registering all handler funcs!")

	for _, reg := range handlerFuncRegistrations {
		err := registerHandlerFunc(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func registerHandlerFunc(reg *handlerFuncRegistration) error {
	regLock.RLock()
	defer regLock.RUnlock()

	http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	return nil
}

func addGlobToDirectoryHandler(dir, requestPath string,
	handler func(http.ResponseWriter, *http.Request)) error {

	var reg *handlerFuncRegistration

	dirHandlerReg, present := handlerFuncRegistrations[dir]
	if !present {
		dirHandler := &directoryHandler{
			DirectoryPath:   dir,
			PatternHandlers: map[string]*handlerFuncRegistration{},
		}

		reg = NewHandlerFuncRegistration(dir,
			func(w http.ResponseWriter, req *http.Request) {
				dirHandler.Handle(w, req)
			})
		reg.Receiver = dirHandler
	}

	dirHandlerReg = handlerFuncRegistrations[dir]
	if dirHandlerReg.Receiver == nil {
		return fmt.Errorf("Cannot add glob to directory handler for %q", dir)
	}

	globReg := &handlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
	}

	err := dirHandlerReg.Receiver.AddGlob(requestPath, globReg)
	if err != nil {
		return err
	}

	return nil
}
