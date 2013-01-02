package goaspen

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
)

var (
	DefaultCharsetDynamic = "utf-8"
	DefaultCharsetStatic  = DefaultCharsetDynamic
	DefaultIndices        = "index.html,index.json,index.txt"

	websites = map[string]*Website{}
)

type Website struct {
	PackageName string
	WwwRoot     string

	CharsetDynamic string
	CharsetStatic  string
	Indices        []string
	ListDirs       bool
	Debug          bool

	server                   *serverContext
	handlerFuncRegistrations map[string]*handlerFuncRegistration
	regLock                  sync.RWMutex
	configured               bool
}

func DeclareWebsite(packageName string) *Website {
	if website, ok := websites[packageName]; ok {
		return website
	}

	newSite := &Website{
		PackageName: packageName,
		Indices:     []string{},

		handlerFuncRegistrations: map[string]*handlerFuncRegistration{},
	}
	websites[packageName] = newSite

	return newSite
}

func (me *Website) NewHTTPResponseWrapper(w http.ResponseWriter, req *http.Request) *HTTPResponseWrapper {
	return &HTTPResponseWrapper{
		website: me,
		w:       w,
		req:     req,

		statusCode: http.StatusOK,
		bodyBytes:  []byte(""),

		contentType:         "text/html",
		contentTypeHandlers: make(map[string]func(*HTTPResponseWrapper)),

		err: nil,
	}
}

func (me *Website) NewHandlerFuncRegistration(requestPath string,
	handler http.HandlerFunc) *handlerFuncRegistration {

	if len(requestPath) < 1 {
		panic(fmt.Errorf("Invalid request path %q", requestPath))
	}

	me.regLock.RLock()
	defer me.regLock.RUnlock()

	me.handlerFuncRegistrations[requestPath] = &handlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
		Receiver:    nil,

		website: me,
	}

	if !strings.HasSuffix(requestPath, "/") && len(path.Ext(requestPath)) == 0 {
		pathGlob := requestPath + ".*"
		me.handlerFuncRegistrations[pathGlob] = &handlerFuncRegistration{
			RequestPath: pathGlob,
			HandlerFunc: handler,
			Receiver:    nil,

			website: me,
		}
	}

	return me.handlerFuncRegistrations[requestPath]
}

func (me *Website) expandAllHandlerFuncRegistrations() error {
	debugf("Expanding all handler func registrations!")

	for _, reg := range me.handlerFuncRegistrations {
		err := me.expandHandlerFuncRegistration(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *Website) expandHandlerFuncRegistration(reg *handlerFuncRegistration) error {
	if path.Ext(reg.RequestPath) == ".*" {
		debugf("Found glob registration %q, adding to directory handler",
			reg.RequestPath)

		err := me.addGlobToDirectoryHandler(me.WwwRoot,
			path.Dir(reg.RequestPath), reg.RequestPath, reg.HandlerFunc)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (me *Website) registerAllHandlerFuncs() error {
	debugf("Registering all handler funcs!")

	for _, reg := range me.handlerFuncRegistrations {
		err := me.registerHandlerFunc(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *Website) registerHandlerFunc(reg *handlerFuncRegistration) error {
	me.regLock.RLock()
	defer me.regLock.RUnlock()

	http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	return nil
}

func (me *Website) addGlobToDirectoryHandler(wwwRoot, dir, requestPath string,
	handler func(http.ResponseWriter, *http.Request)) error {

	var reg *handlerFuncRegistration

	dirHandlerReg, present := me.handlerFuncRegistrations[dir]
	if !present {
		dirHandler := &directoryHandler{
			WwwRoot:         wwwRoot,
			DirectoryPath:   dir,
			PatternHandlers: map[string]*handlerFuncRegistration{},

			website: me,
		}

		reg = me.NewHandlerFuncRegistration(dir,
			func(w http.ResponseWriter, req *http.Request) {
				dirHandler.Handle(w, req)
			})
		reg.Receiver = dirHandler
	}

	dirHandlerReg = me.handlerFuncRegistrations[dir]
	if dirHandlerReg.Receiver == nil {
		return fmt.Errorf("Cannot add glob to directory handler for %q", dir)
	}

	globReg := &handlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
		Receiver:    nil,

		website: me,
	}

	err := dirHandlerReg.Receiver.AddGlob(requestPath, globReg)
	if err != nil {
		return err
	}

	return nil
}

func (me *Website) Configure(serverBind, wwwRoot, charsetDynamic,
	charsetStatic, indices string, debug, listDirs bool) {

	debugf("website.Configure(%q, %q, %q, %q, %q, %v, %v)", serverBind, wwwRoot,
		charsetDynamic, charsetStatic, indices, debug, listDirs)

	me.WwwRoot = wwwRoot
	me.CharsetDynamic = charsetDynamic
	me.CharsetStatic = charsetStatic
	me.ListDirs = listDirs
	me.Debug = debug

	for _, part := range strings.Split(indices, ",") {
		me.Indices = append(me.Indices, strings.TrimSpace(part))
	}

	if me.server == nil {
		me.server = newServerContext(me,
			me.PackageName, serverBind, me.WwwRoot, debug)
	}

	me.configured = true
}

func (me *Website) RunServer() error {
	if !me.configured {
		return fmt.Errorf("Can't run the server when we aren't configured!")
	}

	err := me.expandAllHandlerFuncRegistrations()
	if err != nil {
		return err
	}

	err = me.registerAllHandlerFuncs()
	if err != nil {
		return err
	}

	return me.server.Run()
}
