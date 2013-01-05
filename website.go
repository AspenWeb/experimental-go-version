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

	initialized  = false
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

	server          *serverContext
	pipelineHandler *websitePipelineHandler
	configured      bool
}

type websitePipelineHandler struct {
	website *Website

	httpHandlers             []http.Handler
	handlerFuncRegistrations map[string]*handlerFuncRegistration
	regLock                  sync.RWMutex
}

type websitePatternHandler struct {
	website *Website

	handlerFuncRegistrations map[string]*handlerFuncRegistration
	regLock                  sync.RWMutex
}

type websiteStaticHandler struct {
	website *Website
}

type WebsiteConfigurer struct{}

func EnsureInitialized() *Website {
	if initialized {
		return protoWebsite
	}

	if len(os.Getenv("__GOASPEN_PARENT_PROCESS")) > 0 {
		return protoWebsite
	}

	configScripts := os.Getenv("GOASPEN_CONFIGURATION_SCRIPTS")

	if len(configScripts) > 0 {
		*(&protoWebsite) = loadProtoWebsite(configScripts, protoWebsite)
	}

	*(&initialized) = true
	return protoWebsite
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
	}
	newSite.pipelineHandler = &websitePipelineHandler{
		website: newSite,

		httpHandlers:             []http.Handler{},
		handlerFuncRegistrations: map[string]*handlerFuncRegistration{},
	}
	newSite.pipelineHandler.Add(&websiteStaticHandler{
		website: newSite,
	})
	newSite.pipelineHandler.Add(&websitePatternHandler{
		website: newSite,

		handlerFuncRegistrations: map[string]*handlerFuncRegistration{},
	})

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
	handler http.HandlerFunc, isDir bool) *handlerFuncRegistration {

	return me.pipelineHandler.NewHandlerFuncRegistration(requestPath, handler, isDir)
}

func (me *websitePipelineHandler) NewHandlerFuncRegistration(requestPath string,
	handler http.HandlerFunc, isDir bool) *handlerFuncRegistration {

	debugf("NewHandlerFuncRegistration(%q, <func>)", requestPath)

	if len(requestPath) < 1 {
		panic(fmt.Errorf("Invalid request path %q", requestPath))
	}

	pathBase := path.Base(requestPath)
	pathDir := path.Dir(requestPath)
	isVirtual := vPathPart.MatchString(requestPath)
	debugf("Setting `Virtual` to %v for %q", isVirtual, requestPath)

	requestPathPattern := requestPath
	if isVirtual {
		requestPathPattern = virtualToRegexp(requestPath)
	}

	me.AddHandlerFunc(requestPath, &handlerFuncRegistration{
		RequestPath: requestPathPattern,
		HandlerFunc: handler,
		Virtual:     isVirtual,

		website: me.website,
	})

	debugf("Checking if %q matches any of %v", pathBase, me.website.Indices)

	for _, idx := range me.website.Indices {
		if pathBase == idx {
			debugf("Registering %q with same handler as %q", pathDir, pathBase)
			me.AddHandlerFunc(pathDir, &handlerFuncRegistration{
				RequestPath: pathDir,
				HandlerFunc: handler,
				Virtual:     isVirtual,

				website: me.website,
			})
		}
	}

	if !isDir && !strings.HasSuffix(requestPath, "/") && len(path.Ext(requestPath)) == 0 {
		debugf("Registering %q as a negotiated simplate", requestPath)

		pathRegexp := requestPathPattern + "\\.[^\\.]+"

		me.AddHandlerFunc(pathRegexp, &handlerFuncRegistration{
			RequestPath: pathRegexp,
			HandlerFunc: handler,
			Negotiated:  true,
			Virtual:     isVirtual,

			website: me.website,
		})
	}

	return me.HandlerFuncAt(requestPath)
}

func virtualToRegexp(requestPath string) string {
	return vPathPart.ReplaceAllString(requestPath, "(?P<$1>[a-zA-Z_][-a-zA-Z0-9_]*)")
}

func findHighestNormalDir(requestPath string) string {
	if strings.Contains(requestPath, "%") {
		parts := strings.SplitN(requestPath, "%", 2)
		if len(parts[0]) > 1 {
			return parts[0]
		}
	}

	parts := strings.SplitN(requestPath, "\\", 2)
	if len(parts[0]) > 1 && strings.HasSuffix(parts[0], "/") {
		return parts[0]
	}

	return "/"
}

func (me *websitePipelineHandler) expandAllHandlerFuncRegistrations() error {
	debugf("Expanding all handler func registrations!")

	for _, reg := range me.handlerFuncRegistrations {
		err := me.expandHandlerFuncRegistration(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *websitePipelineHandler) expandHandlerFuncRegistration(reg *handlerFuncRegistration) error {
	destDir := findHighestNormalDir(reg.RequestPath)

	if reg.Virtual || reg.Negotiated {
		debugf("Adding negotiated or virtual path registration %q to "+
			"directory handler at %q", reg.RequestPath, destDir)

		err := me.addRegexpToDirectoryHandler(me.website.WwwRoot,
			destDir, reg.RequestPath, reg.HandlerFunc)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (me *websitePipelineHandler) registerAllHandlerFuncs() error {
	debugf("Registering all handler funcs!")

	for _, reg := range me.handlerFuncRegistrations {
		err := me.registerHandlerFunc(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (me *websitePipelineHandler) registerHandlerFunc(reg *handlerFuncRegistration) error {
	me.regLock.RLock()
	defer me.regLock.RUnlock()

	http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	return nil
}

func (me *websitePipelineHandler) addRegexpToDirectoryHandler(wwwRoot, dir, requestPath string,
	handler http.HandlerFunc) error {

	var reg *handlerFuncRegistration

	dirHandlerReg, present := me.handlerFuncRegistrations[dir]
	if !present {
		dirHandler := &directoryHandler{
			WwwRoot:         wwwRoot,
			DirectoryPath:   dir,
			PatternHandlers: map[string]*handlerFuncRegistration{},

			website: me.website,
		}

		debugf("Creating new handler func for dir %q", dir)
		reg = me.NewHandlerFuncRegistration(dir,
			func(w http.ResponseWriter, req *http.Request) {
				dirHandler.Handle(w, req)
			}, true)
		reg.Receiver = dirHandler
	}

	if present && dirHandlerReg.Receiver == nil {
		debugf("Already non-dir handler %q for %q, so going up a level",
			dirHandlerReg.RequestPath, requestPath)
		return me.addRegexpToDirectoryHandler(wwwRoot,
			path.Dir(dir), requestPath, handler)
	}

	dirHandlerReg = me.handlerFuncRegistrations[dir]
	if dirHandlerReg.Receiver == nil {
		return fmt.Errorf("Cannot add regexp to directory handler for %q", dir)
	}

	reReg := &handlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,

		website: me.website,
	}

	err := dirHandlerReg.Receiver.AddRegexp(requestPath, reReg)
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

func (me *websitePipelineHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	panic(fmt.Errorf("NOT IMPLEMENTED"))
}

func (me *websitePipelineHandler) Add(h http.Handler) {
	me.httpHandlers = append(me.httpHandlers, h)
}

func (me *websitePipelineHandler) AddHandlerFunc(requestPath string,
	r *handlerFuncRegistration) {

	me.regLock.RLock()
	defer me.regLock.RUnlock()

	me.handlerFuncRegistrations[requestPath] = r
}

func (me *websitePipelineHandler) HandlerFuncAt(requestPath string) *handlerFuncRegistration {
	if r, ok := me.handlerFuncRegistrations[requestPath]; ok {
		return r
	}

	return nil
}

func (me *websitePatternHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	panic(fmt.Errorf("NOT IMPLEMENTED"))
}

func (me *websiteStaticHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// TODO remember to use http.ServeFile!
	panic(fmt.Errorf("NOT IMPLEMENTED"))
}

func (me *Website) RunServer() error {
	if !me.configured {
		return fmt.Errorf("Can't run the server when we aren't configured!")
	}

	err := me.pipelineHandler.expandAllHandlerFuncRegistrations()
	if err != nil {
		return err
	}

	err = me.pipelineHandler.registerAllHandlerFuncs()
	if err != nil {
		return err
	}

	return me.server.Run()
}

func (me *Website) DebugNewRequest(req *http.Request) {
	debugf("Handling new request %+v", req.URL)
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
