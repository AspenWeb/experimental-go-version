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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
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

func (me *DirectoryHandler) updateNegType(req *http.Request, filename string) {
	mediaType := mime.TypeByExtension(path.Ext(filename))
	if len(mediaType) == 0 {
		mediaType = "text/html" // FIXME get default from config
	}

	req.Header.Set(internalAcceptHeader, mediaType)
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

func ExpandAllHandlerFuncRegistrations() error {
	debugf("Expanding all handler func registrations!")

	for _, reg := range HandlerFuncRegistrations {
		err := expandHandlerFuncRegistration(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func expandHandlerFuncRegistration(reg *HandlerFuncRegistration) error {
	if path.Ext(reg.RequestPath) == ".*" {
		debugf("Found glob registration %q, adding to directory handler",
			reg.RequestPath)

		err := AddGlobToDirectoryHandler(path.Dir(reg.RequestPath),
			reg.RequestPath, reg.HandlerFunc)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func RegisterAllHandlerFuncs() error {
	debugf("Registering all handler funcs!")

	for _, reg := range HandlerFuncRegistrations {
		err := registerHandlerFunc(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func registerHandlerFunc(reg *HandlerFuncRegistration) error {
	regLock.RLock()
	defer regLock.RUnlock()

	http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	return nil
}

func AddGlobToDirectoryHandler(dir, requestPath string,
	handler func(http.ResponseWriter, *http.Request)) error {

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
		return errors.New(msg)
	}

	globReg := &HandlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
	}
	err := dirHandlerReg.Receiver.AddGlob(requestPath, globReg)
	if err != nil {
		return err
	}

	return nil
}
