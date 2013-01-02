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
	DefaultIndices        = "index.html, index.txt, index.json"

	apps = map[string]*App{}
)

type App struct {
	PackageName string
	WwwRoot     string

	CharsetDynamic string
	CharsetStatic  string
	Indices        []string
	ListDirs       bool

	server                   *serverContext
	handlerFuncRegistrations map[string]*handlerFuncRegistration
	regLock                  sync.RWMutex
	configured               bool
}

func DeclareApp(packageName string) *App {
	if app, ok := apps[packageName]; ok {
		return app
	}

	newApp := &App{
		PackageName: packageName,
		Indices:     []string{},

		handlerFuncRegistrations: map[string]*handlerFuncRegistration{},
	}
	apps[packageName] = newApp

	return newApp
}

func LookupApp(packageName string) *App {
	if app, ok := apps[packageName]; ok {
		return app
	}

	panic(fmt.Errorf("There is no app registered for package %q!", packageName))
}

func (me *App) NewHTTPResponseWrapper(w http.ResponseWriter, req *http.Request) *HTTPResponseWrapper {
	return &HTTPResponseWrapper{
		app: me,
		w:   w,
		req: req,

		statusCode: http.StatusOK,
		bodyBytes:  []byte(""),

		contentType:         "text/html",
		contentTypeHandlers: make(map[string]func(*HTTPResponseWrapper)),

		err: nil,
	}
}

func (me *App) NewHandlerFuncRegistration(requestPath string,
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

		app: me,
	}

	if !strings.HasSuffix(requestPath, "/") && len(path.Ext(requestPath)) == 0 {
		pathGlob := requestPath + ".*"
		me.handlerFuncRegistrations[pathGlob] = &handlerFuncRegistration{
			RequestPath: pathGlob,
			HandlerFunc: handler,
			Receiver:    nil,

			app: me,
		}
	}

	return me.handlerFuncRegistrations[requestPath]
}

func (me *App) expandAllHandlerFuncRegistrations() error {
	debugf("Expanding all handler func registrations!")

	for _, reg := range me.handlerFuncRegistrations {
		err := me.expandHandlerFuncRegistration(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *App) expandHandlerFuncRegistration(reg *handlerFuncRegistration) error {
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

func (me *App) registerAllHandlerFuncs() error {
	debugf("Registering all handler funcs!")

	for _, reg := range me.handlerFuncRegistrations {
		err := me.registerHandlerFunc(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *App) registerHandlerFunc(reg *handlerFuncRegistration) error {
	me.regLock.RLock()
	defer me.regLock.RUnlock()

	http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	return nil
}

func (me *App) addGlobToDirectoryHandler(wwwRoot, dir, requestPath string,
	handler func(http.ResponseWriter, *http.Request)) error {

	var reg *handlerFuncRegistration

	dirHandlerReg, present := me.handlerFuncRegistrations[dir]
	if !present {
		dirHandler := &directoryHandler{
			WwwRoot:         wwwRoot,
			DirectoryPath:   dir,
			PatternHandlers: map[string]*handlerFuncRegistration{},

			app: me,
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

		app: me,
	}

	err := dirHandlerReg.Receiver.AddGlob(requestPath, globReg)
	if err != nil {
		return err
	}

	return nil
}

func (me *App) Configure(serverBind, wwwRoot, charsetDynamic,
	charsetStatic, indices string, debug, listDirs bool) {

	debugf("app.Configure(%q, %q, %q, %q, %q, %v, %v)", serverBind, wwwRoot,
		charsetDynamic, charsetStatic, indices, debug, listDirs)

	me.WwwRoot = wwwRoot
	me.CharsetDynamic = charsetDynamic
	me.CharsetStatic = charsetStatic
	me.ListDirs = listDirs

	for _, part := range strings.Split(indices, ",") {
		me.Indices = append(me.Indices, strings.TrimSpace(part))
	}

	if me.server == nil {
		me.server = newServerContext(me,
			me.PackageName, serverBind, me.WwwRoot, debug)
	}

	me.configured = true
}

func (me *App) RunServer() error {
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
