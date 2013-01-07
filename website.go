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
	"sort"
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
	String() string
}

type websitePipelineHandler struct {
	w *Website

	nh pipelineHandler
	r  map[string]*handlerFuncRegistration
	l  sync.RWMutex

	patternHandler  *websitePatternHandler
	strMatchHandler *websiteStringMatchHandler
}

type websiteStringMatchHandler struct {
	w *Website

	nh pipelineHandler
	r  map[string]*handlerFuncRegistration
	l  sync.RWMutex
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
	strMatchHandler := &websiteStringMatchHandler{
		w: newSite,
		r: map[string]*handlerFuncRegistration{},

		nh: patternHandler,
	}
	ph := &websitePipelineHandler{
		w: newSite,

		nh: strMatchHandler,
	}

	ph.patternHandler = patternHandler
	ph.strMatchHandler = strMatchHandler
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

	isVirtual := vPathPart.MatchString(requestPath)
	debugf("Setting `Virtual` to %v for %q", isVirtual, requestPath)

	if isVirtual || simplateType == SimplateTypeNegotiated {
		return me.patternHandler.newHandlerFuncRegistration(requestPath,
			simplateType, handler, isDir, isVirtual)
	}

	return me.strMatchHandler.newHandlerFuncRegistration(requestPath,
		simplateType, handler, isDir)
}

func (me *websitePatternHandler) newHandlerFuncRegistration(requestPath,
	simplateType string, handler http.HandlerFunc,
	isDir, isVirtual bool) *handlerFuncRegistration {

	if len(requestPath) < 1 {
		panic(fmt.Errorf("Invalid request path %q", requestPath))
	}

	requestPathPattern := requestPath
	if isVirtual {
		requestPathPattern = virtualToRegexp(requestPath)
	}

	if simplateType == SimplateTypeNegotiated {
		pathRegexp := requestPathPattern + "\\.[^\\.]+"
		debugf("Registering %q as a negotiated simplate", pathRegexp)

		me.AddHandlerFuncReg(requestPath, &handlerFuncRegistration{
			RequestPath: pathRegexp,
			HandlerFunc: handler,
			Negotiated:  true,
			Virtual:     isVirtual,
			Regexp:      true,

			w: me.w,
		})

		return me.HandlerFuncAt(requestPath)
	}

	me.AddHandlerFuncReg(requestPath, &handlerFuncRegistration{
		RequestPath: requestPathPattern,
		HandlerFunc: handler,
		Virtual:     isVirtual,
		Negotiated:  simplateType == SimplateTypeNegotiated,
		Regexp:      isVirtual,

		w: me.w,
	})

	return me.HandlerFuncAt(requestPath)
}

func (me *websiteStringMatchHandler) newHandlerFuncRegistration(requestPath,
	simplateType string, handler http.HandlerFunc,
	isDir bool) *handlerFuncRegistration {

	if simplateType == SimplateTypeNegotiated {
		debugf("Ignoring negotiated simplate registration for %q", requestPath)
		return nil
	}

	pathBase := path.Base(requestPath)
	pathDir := path.Dir(requestPath)

	debugf("Checking if %q matches any of %v", pathBase, me.w.Indices)

	var reg *handlerFuncRegistration

	for _, idx := range me.w.Indices {
		if pathBase == idx {
			reqPath := pathDir + "/"

			reg = &handlerFuncRegistration{
				RequestPath: reqPath,
				HandlerFunc: handler,

				w: me.w,
			}

			debugf("Registering %q with same handler as %q", reqPath, pathBase)
			me.AddHandlerFuncReg(pathDir, &handlerFuncRegistration{
				RequestPath: pathDir,
				HandlerFunc: func(w http.ResponseWriter, req *http.Request) {
					h := http.RedirectHandler(reqPath, http.StatusMovedPermanently)
					h.ServeHTTP(w, req)
				},

				w: me.w,
			})
			me.AddHandlerFuncReg(reqPath, reg)
		}
	}

	return reg
}

func virtualToRegexp(requestPath string) string {
	return vPathPart.ReplaceAllString(requestPath, vPathPartRep)
}

func (me *websitePipelineHandler) registerSpecialCases() {
	idxPath := "/" + SiteIndexFilename
	debugf("Registering special case of %q -> 404", idxPath)
	me.strMatchHandler.AddHandlerFuncReg(idxPath, &handlerFuncRegistration{
		RequestPath: idxPath,
		HandlerFunc: serve404,
	})
}

func (me *websitePipelineHandler) registerSelfAtRoot() {
	debugf(`Registering pipeline handler at "/"`)
	http.Handle("/", me)
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

	sortedIndices := make([]string, len(me.Indices))
	copy(sortedIndices, me.Indices)
	sort.Strings(sortedIndices)

	for _, part := range strings.Split(indices, ",") {
		trimmed := strings.TrimSpace(part)
		if sort.SearchStrings(sortedIndices, trimmed) > -1 {
			debugf("*NOT* appending duplicate index name %q into %v",
				trimmed, me.Indices)
		} else {
			debugf("Adding index name %q to %v", trimmed, me.Indices)
			me.Indices = append(me.Indices, trimmed)
		}
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

	h := me.NextHandler()
	if h != nil {
		debugf("Pipeline handler sending %q to %s", req.URL.Path, h)
		h.ServeHTTP(w, req)
	}
}

func (me *websitePipelineHandler) String() string {
	return fmt.Sprintf("*websitePipelineHandler{"+
		"patternHandler: %s, "+
		"strMatchHandler: %s, "+
		"r: %+v}", me.patternHandler, me.strMatchHandler, me.r)
}

func (me *websitePipelineHandler) injectCustomHeaders(req *http.Request) {
	me.updateNegType(req, req.URL.Path)
	req.Header.Set("X-GoAspen-PackageName", me.w.PackageName)
	req.Header.Set("X-GoAspen-WwwRoot", me.w.WwwRoot)
	req.Header.Set("X-GoAspen-CharsetStatic", me.w.CharsetStatic)
	req.Header.Set("X-GoAspen-CharsetDynamic", me.w.CharsetDynamic)
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

func (me *websitePatternHandler) AddHandlerFuncReg(requestPath string,
	r *handlerFuncRegistration) {

	if r.Negotiated && !r.Regexp {
		debugf("Intercepting non-regexp negotiated registration for %q, "+
			"replacing with 404 handler", requestPath)
		r = &handlerFuncRegistration{
			RequestPath: requestPath,
			HandlerFunc: serve404,

			w: me.w,
		}
	}

	debugf("Adding handler func registration for %q: %+v", requestPath, r)

	me.l.RLock()
	defer me.l.RUnlock()

	if _, ok := me.r[requestPath]; ok {
		debugf("Ignoring additional registration for %q", requestPath)
		return
	}

	me.c[requestPath] = regexp.MustCompile(r.RequestPath)

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
	debugf("Pattern handler looking for registration that matches %q", req.URL.Path)

	// Loop through the non-regexp request paths and their registrations.
	for requestPath, reg := range me.r {
		// Get the compiled regexp for the registered request path, and if the
		// incoming request URL Path matches, call the regitration's HandlerFunc.
		re := me.c[requestPath]
		if re.MatchString(req.URL.Path) {
			reg.HandlerFunc(w, req)
			return
		}
	}

	h := me.NextHandler()
	if h != nil {
		debugf("Pattern handler falling through to %s", h)
		h.ServeHTTP(w, req)
		return
	}

	debugf("Pattern handler falling through to 404 because next handler is %v", h)
	serve404(w, req)
}

func (me *websitePatternHandler) String() string {
	return fmt.Sprintf("*websitePatternHandler{r: %v}", me.r)
}

func (me *websitePatternHandler) findVpathRegexp(vPathString string) *regexp.Regexp {
	if re, ok := me.c[vPathString]; ok {
		return re
	}

	return nil
}

func (me *websiteStringMatchHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	reg := me.match(req.URL.Path)

	if reg != nil {
		reg.HandlerFunc(w, req)
		return
	}

	me.NextHandler().ServeHTTP(w, req)
	return
}

func (me *websiteStringMatchHandler) String() string {
	return fmt.Sprintf("*websiteStringMatchHandler{r: %v}", me.r)
}

func (me *websiteStringMatchHandler) NextHandler() pipelineHandler {
	return me.nh
}

func (me *websiteStringMatchHandler) AddHandlerFuncReg(requestPath string,
	reg *handlerFuncRegistration) {

	me.l.RLock()
	defer me.l.RUnlock()

	me.r[requestPath] = reg
}

func (me *websiteStringMatchHandler) match(requestPath string) *handlerFuncRegistration {
	var h *handlerFuncRegistration

	n := 0
	for k, v := range me.r {
		if !pathMatch(k, requestPath) {
			continue
		}

		if h == nil || len(k) > n {
			n = len(k)
			h = v
		}
	}

	return h
}

func (me *Website) UpdateContextFromVirtualPaths(ctx *map[string]interface{},
	requestPath, vPathString string) {

	// FIXME Demeter!
	vPath := me.ph.patternHandler.findVpathRegexp(vPathString)
	if vPath == nil {
		debugf("No matching regexp for vpath %q.  Not updating context.",
			vPathString)
		return
	}

	matches := vPath.FindStringSubmatch(requestPath)
	if len(matches) == 0 {
		debugf("Request path %q does not match %q.  Not updating context.",
			requestPath, vPath.String())
		return
	}

	realCtx := *ctx

	names := vPath.SubexpNames()
	for i, match := range matches {
		if len(names[i]) > 0 {
			realCtx[names[i]] = match
		}
	}
}

func (me *Website) RunServer() error {
	if !me.configured {
		return fmt.Errorf("Can't run the server when we aren't configured!")
	}

	me.ph.registerSpecialCases()
	me.ph.registerSelfAtRoot()

	if isDebug {
		debugf("Website about to run server with pipeline:\n\t%s", me.ph)

		debugf("String matches registered:")
		for m, _ := range me.ph.strMatchHandler.r {
			debugf("    %s", m)
		}

		debugf("Patterns registered:")
		for _, re := range me.ph.patternHandler.c {
			debugf("    %s", re)
		}
	}

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
