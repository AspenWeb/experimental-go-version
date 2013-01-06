package goaspen

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
)

var (
	DefaultCharsetDynamic = "utf-8"
	DefaultCharsetStatic  = DefaultCharsetDynamic
	DefaultContentType    = "application/octet-stream"
	DefaultIndicesArray   = []string{"index.html", "index.json", "index.txt"}
	DefaultIndices        = strings.Join(DefaultIndicesArray, ",")
	DefaultConfig         = &WebsiteConfigurer{}

	initialized  = false
	websites     = map[string]*Website{}
	protoWebsite = &Website{
		PackageName: DefaultGenPackage,
		WwwRoot:     ".",

		CharsetDynamic:     DefaultCharsetDynamic,
		CharsetStatic:      DefaultCharsetStatic,
		DefaultContentType: DefaultContentType,
		Indices:            DefaultIndicesArray,
		ListDirs:           false,
		Debug:              false,
	}
)

type Website struct {
	PackageName string
	WwwRoot     string

	CharsetDynamic     string
	CharsetStatic      string
	DefaultContentType string
	Indices            []string
	ListDirs           bool
	Debug              bool

	configured bool

	s  *serverContext
	ph *websitePipelineHandler
}

type pipelineHandler interface {
	http.Handler
	NextHandler() pipelineHandler
}

type websitePipelineHandler struct {
	w *Website

	nh             pipelineHandler
	patternHandler *websitePatternHandler
}

type websitePatternHandler struct {
	w *Website

	nh pipelineHandler
	r  map[string]*handlerFuncRegistration
	c  map[string]*regexp.Regexp
	l  sync.RWMutex
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
	if w, ok := websites[packageName]; ok {
		return w
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
	staticHandler := &websiteStaticHandler{
		w: newSite,
	}
	patternHandler := &websitePatternHandler{
		w: newSite,

		r:  map[string]*handlerFuncRegistration{},
		c:  map[string]*regexp.Regexp{},
		nh: staticHandler,
	}
	ph := &websitePipelineHandler{
		w: newSite,

		nh: patternHandler,
	}

	ph.patternHandler = patternHandler
	newSite.ph = ph

	websites[packageName] = newSite

	return newSite
}

func loadProtoWebsite(configScripts string, proto *Website) *Website {
	var err error

	scripts := strings.Split(configScripts, ",")
	w := proto

	for _, script := range scripts {
		w, err = loadWebsiteFromScript(strings.TrimSpace(script), w)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goaspen: CONFIG ERROR: %v\n", err)
		}
	}

	return w
}

func loadWebsiteFromScript(script string, w *Website) (*Website, error) {
	encoded, err := json.Marshal(w)
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

	err = json.Unmarshal(outbytes, w)
	if err != nil {
		cmd.Wait()
		return nil, err
	}

	debugf("Loaded website from config script %v: %+v", script, w)

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "goaspen: CONFIG ERROR: %v\n", err)
	}

	return w, nil
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

func (me *Website) RegisterSimplate(simplateType, siteRoot, requestPath string,
	handler http.HandlerFunc) *handlerFuncRegistration {

	return me.ph.NewHandlerFuncRegistration(requestPath,
		simplateType, handler, false)
}

func (me *websitePipelineHandler) NewHandlerFuncRegistration(requestPath,
	simplateType string, handler http.HandlerFunc, isDir bool) *handlerFuncRegistration {

	debugf("NewHandlerFuncRegistration(%q, %q, <func>, %v)", requestPath, simplateType, isDir)

	return me.patternHandler.newHandlerFuncRegistration(requestPath,
		simplateType, handler, isDir)
}

func (me *websitePatternHandler) newHandlerFuncRegistration(requestPath,
	simplateType string, handler http.HandlerFunc, isDir bool) *handlerFuncRegistration {

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
		// All virtual are regexp, but not all regexp are virtual
		Regexp: isVirtual,

		w: me.w,
	})

	debugf("Checking if %q matches any of %v", pathBase, me.w.Indices)

	for _, idx := range me.w.Indices {
		if pathBase == idx {
			// net/http will automatically add a redirect from "pathDir" if
			// we register with a trailing slash.
			reqPath := pathDir + "/"
			debugf("Registering %q with same handler as %q", reqPath, pathBase)

			reg := &handlerFuncRegistration{
				RequestPath: reqPath,
				HandlerFunc: handler,
				Virtual:     isVirtual,
				Regexp:      isVirtual,

				w: me.w,
			}

			me.AddHandlerFunc(reqPath, reg)
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
			Regexp:      true,

			w: me.w,
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

func (me *websitePipelineHandler) registerAllHandlerFuncs() {
	debugf("Registering handler funcs!")

	me.patternHandler.registerValidHandlerFuncs()
}

func (me *websitePipelineHandler) registerSelfAtRoot() {
	debugf(`Registering pipeline handler at "/"`)
	http.Handle("/", me)
}

func (me *websitePatternHandler) registerValidHandlerFuncs() {
	me.l.RLock()
	defer me.l.RUnlock()

	for _, reg := range me.r {
		registerHandlerFuncIfValid(reg)
	}
}

func registerHandlerFuncIfValid(reg *handlerFuncRegistration) {
	if !reg.Regexp {
		http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	} else {
		debugf("NOT registering handler func for %q with net/http", reg.RequestPath)
	}
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

	if me.s == nil {
		me.s = newServerContext(me,
			me.PackageName, serverBind, me.WwwRoot, debug)
	}

	me.configured = true
}

func (me *websitePipelineHandler) NextHandler() pipelineHandler {
	return me.nh
}

func (me *websitePipelineHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	me.injectCustomHeaders(req)
	debugf("Pipeline handler sending %q to pattern handler", req.URL.Path)
	me.patternHandler.ServeHTTP(w, req)
}

func (me *websitePipelineHandler) injectCustomHeaders(req *http.Request) {
	me.updateNegType(req, req.URL.Path)
	req.Header.Set("X-GoAspen-PackageName", me.w.PackageName)
	req.Header.Set("X-GoAspen-WwwRoot", me.w.WwwRoot)
}

func (me *websitePipelineHandler) updateNegType(req *http.Request, filename string) {
	mediaType := mime.TypeByExtension(path.Ext(filename))
	if len(mediaType) == 0 {
		mediaType = me.w.DefaultContentType
	}

	req.Header.Set(internalAcceptHeader, mediaType)
}

func (me *websitePatternHandler) NextHandler() pipelineHandler {
	return me.nh
}

func (me *websitePatternHandler) AddHandlerFunc(requestPath string,
	r *handlerFuncRegistration) {

	debugf("Adding handler func registration for %q: %+v", requestPath, r)

	me.l.RLock()
	defer me.l.RUnlock()

	if _, ok := me.r[requestPath]; ok {
		debugf("Ignoring additional registration for %q", requestPath)
		return
	}

	me.c[requestPath] = regexp.MustCompile(requestPath)

	debugf("Setting handler for %q", requestPath)
	me.r[requestPath] = r
}

func (me *websitePatternHandler) HandlerFuncAt(requestPath string) *handlerFuncRegistration {
	if r, ok := me.r[requestPath]; ok {
		return r
	}

	return nil
}

func (me *websitePatternHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	reqPathBytes := []byte(req.URL.Path)

	debugf("Pattern handler looking for registration that matches %q", req.URL.Path)

	for requestPath, reg := range me.r {
		re := me.c[requestPath]
		if re.Match(reqPathBytes) {
			reg.HandlerFunc(w, req)
			return
		}
	}

	h := me.NextHandler()
	if h != nil {
		debugf("Pattern handler falling through to next handler")
		h.ServeHTTP(w, req)
		return
	}

	debugf("Pattern handler falling through to 404 because next handler is %v", h)
	serve404(w, req, me.w.CharsetDynamic)
}

func (me *Website) RunServer() error {
	if !me.configured {
		return fmt.Errorf("Can't run the server when we aren't configured!")
	}

	me.ph.registerAllHandlerFuncs()
	me.ph.registerSelfAtRoot()

	return me.s.Run()
}

func (me *Website) DebugNewRequest(simplatePath string, req *http.Request) {
	debugf("%q handling new request %q", simplatePath, req.URL)
}

func (me *WebsiteConfigurer) Load(r io.Reader) (*Website, error) {
	if r == nil {
		r = os.Stdin
	}

	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	w := &Website{}

	err = json.Unmarshal(raw, w)
	if err != nil {
		return nil, err
	}

	return w, nil
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
	w, err := me.Load(r)
	if err != nil {
		panic(err)
	}

	return w
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

func MustDumpWebsite(w *Website) {
	DefaultConfig.MustDump(w, os.Stdout)
}
