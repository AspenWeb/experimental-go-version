package goaspen

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

var (
	DefaultCharsetDynamic = "utf-8"
	DefaultCharsetStatic  = DefaultCharsetDynamic
	DefaultIndicesArray   = []string{"index.html", "index.json", "index.txt"}
	DefaultIndices        = strings.Join(DefaultIndicesArray, ",")
	DefaultConfig         = &WebsiteConfigurer{}

	websites     = map[string]*Website{}
	protoWebsite = &Website{
		PackageName: DefaultGenPackage,
		WwwRoot:     ".",

		CharsetDynamic: DefaultCharsetDynamic,
		CharsetStatic:  DefaultCharsetStatic,
		Indices:        DefaultIndicesArray,
		ListDirs:       false,
		Debug:          false,
	}
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

type WebsiteConfigurer struct {
}

func init() {
	if len(os.Getenv("__GOASPEN_PARENT_PROCESS")) > 0 {
		return
	}

	configScripts := os.Getenv("GOASPEN_CONFIGURATION_SCRIPTS")

	if len(configScripts) > 0 {
		protoWebAddr := &protoWebsite
		*protoWebAddr = loadProtoWebsite(configScripts, protoWebsite)
	}
}

func DeclareWebsite(packageName string) *Website {
	if website, ok := websites[packageName]; ok {
		return website
	}

	newSite := &Website{
		PackageName: packageName,
		WwwRoot:     protoWebsite.WwwRoot,

		CharsetDynamic: protoWebsite.CharsetDynamic,
		CharsetStatic:  protoWebsite.CharsetStatic,
		Indices:        protoWebsite.Indices,
		ListDirs:       protoWebsite.ListDirs,
		Debug:          protoWebsite.Debug,

		handlerFuncRegistrations: map[string]*handlerFuncRegistration{},
	}
	websites[packageName] = newSite

	return newSite
}

func loadProtoWebsite(configScripts string, proto *Website) *Website {
	var err error

	scripts := strings.Split(configScripts, ",")
	website := proto

	for _, script := range scripts {
		website, err = loadWebsiteFromScript(strings.TrimSpace(script), website)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goaspen: CONFIG ERROR: %v\n", err)
		}
	}

	return website
}

func loadWebsiteFromScript(script string, website *Website) (*Website, error) {
	encoded, err := json.Marshal(website)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("go", "run", script)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("__GOASPEN_PARENT_PROCESS=%d", os.Getpid()))

	cmd.Stderr = os.Stderr

	inbuf, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	outbuf, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	_, err = inbuf.Write(encoded)
	if err != nil {
		cmd.Wait()
		return nil, err
	}

	err = inbuf.Close()
	if err != nil {
		cmd.Wait()
		return nil, err
	}

	outbytes, err := ioutil.ReadAll(outbuf)
	if err != nil {
		cmd.Wait()
		return nil, err
	}

	err = json.Unmarshal(outbytes, website)
	if err != nil {
		cmd.Wait()
		return nil, err
	}

	debugf("Loaded website from config script %v: %+v", script, website)

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "goaspen: CONFIG ERROR: %v\n", err)
	}

	return website, nil
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

	debugf("NewHandlerFuncRegistration(%q, <func>)", requestPath)

	if len(requestPath) < 1 {
		panic(fmt.Errorf("Invalid request path %q", requestPath))
	}

	me.regLock.RLock()
	defer me.regLock.RUnlock()

	pathBase := path.Base(requestPath)
	pathDir := path.Dir(requestPath)

	me.handlerFuncRegistrations[requestPath] = &handlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
		Receiver:    nil,

		website: me,
	}

	debugf("Checking if %q matches any of %v", pathBase, me.Indices)

	for _, idx := range me.Indices {
		if pathBase == idx {
			debugf("Registering %q with same handler as %q", pathDir, pathBase)
			me.handlerFuncRegistrations[pathDir] = &handlerFuncRegistration{
				RequestPath: pathDir,
				HandlerFunc: handler,
				Receiver:    nil,

				website: me,
			}
		}
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

func (me *WebsiteConfigurer) Load(r io.Reader) (*Website, error) {
	if r == nil {
		r = os.Stdin
	}

	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	website := &Website{}

	err = json.Unmarshal(raw, website)
	if err != nil {
		return nil, err
	}

	return website, nil
}

func (me *WebsiteConfigurer) Dump(website *Website, w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}

	encoded, err := json.Marshal(website)
	if err != nil {
		return err
	}

	_, err = w.Write(encoded)
	if err != nil {
		return err
	}

	return nil
}

func (me *WebsiteConfigurer) MustLoad(r io.Reader) *Website {
	website, err := me.Load(r)
	if err != nil {
		panic(err)
	}

	return website
}

func (me *WebsiteConfigurer) MustDump(website *Website, w io.Writer) {
	err := me.Dump(website, w)
	if err != nil {
		panic(err)
	}
}

func MustLoadWebsite() *Website {
	return DefaultConfig.MustLoad(os.Stdin)
}

func MustDumpWebsite(website *Website) {
	DefaultConfig.MustDump(website, os.Stdout)
}
