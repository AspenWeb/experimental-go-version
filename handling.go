package goaspen

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
)

var (
	HandlerFuncRegistrations = map[string]*HandlerFuncRegistration{}

	regLock sync.RWMutex
)

type HandlerFuncRegistration struct {
	RequestPath string
	HandlerFunc func(http.ResponseWriter, *http.Request)
	Receiver    *DirectoryHandler
}

type DirectoryHandler struct {
	SiteRoot        string
	DirectoryPath   string
	PatternHandlers map[string]*HandlerFuncRegistration
}

func (me *DirectoryHandler) Handle(w http.ResponseWriter, req *http.Request) {
	debugf("Handling directory response for %q", req.URL.Path)

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
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		w.Write(http404Response)
	}
}

func (me *DirectoryHandler) AddGlob(pathGlob string,
	reg *HandlerFuncRegistration) error {

	debugf("Adding glob %q to pattern handlers for %q", pathGlob, me.DirectoryPath)
	me.PatternHandlers[pathGlob] = reg
	return nil
}

func (me *DirectoryHandler) ServeStatic(w http.ResponseWriter, req *http.Request) error {
	return errors.New("Not implemented, so pretending nothing is here!")
}

func NewHandlerFuncRegistration(requestPath string,
	handler func(http.ResponseWriter, *http.Request)) *HandlerFuncRegistration {

	if len(requestPath) < 1 {
		panic(errors.New(fmt.Sprintf("Invalid request path %q", requestPath)))
	}

	regLock.RLock()
	defer regLock.RUnlock()

	HandlerFuncRegistrations[requestPath] = &HandlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
	}

	if !strings.HasSuffix(requestPath, "/") && len(path.Ext(requestPath)) == 0 {
		pathGlob := requestPath + ".*"
		HandlerFuncRegistrations[pathGlob] = &HandlerFuncRegistration{
			RequestPath: pathGlob,
			HandlerFunc: handler,
		}
	}

	return HandlerFuncRegistrations[requestPath]
}

func RegisterAllHandlerFuncs() error {
	debugf("Registering all handler funcs!")
	regLock.RLock()
	defer regLock.RUnlock()

	for _, reg := range HandlerFuncRegistrations {
		if path.Ext(reg.RequestPath) == ".*" {
			debugf("Found glob registration %q, adding to directory handler",
				reg.RequestPath)

			newRegistrations, err := AddGlobToDirectoryHandler(path.Dir(reg.RequestPath),
				reg.RequestPath, reg.HandlerFunc)
			if err != nil {
				return err
			}

			for _, newReg := range newRegistrations {
				http.HandleFunc(newReg.RequestPath, newReg.HandlerFunc)
			}
			continue
		}

		http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	}

	return nil
}

func AddGlobToDirectoryHandler(dir, requestPath string,
	handler func(http.ResponseWriter, *http.Request)) ([]*HandlerFuncRegistration, error) {

	var reg *HandlerFuncRegistration

	dirHandlerReg, present := HandlerFuncRegistrations[dir]
	if !present {
		dirHandler := &DirectoryHandler{
			DirectoryPath:   dir,
			PatternHandlers: map[string]*HandlerFuncRegistration{},
		}

		reg = NewHandlerFuncRegistration(dir,
			func(w http.ResponseWriter, req *http.Request) {
				dirHandler.Handle(w, req)
			})
		reg.Receiver = dirHandler
	}

	dirHandlerReg = HandlerFuncRegistrations[dir]
	if dirHandlerReg.Receiver == nil {
		msg := fmt.Sprintf("Cannot add glob to directory handler for %q", dir)
		return nil, errors.New(msg)
	}

	globReg := &HandlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
	}
	err := dirHandlerReg.Receiver.AddGlob(requestPath, globReg)
	if err != nil {
		return nil, err
	}

	return []*HandlerFuncRegistration{dirHandlerReg, globReg}, nil
}
