package goaspen

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jteeuwen/go-pkg-optarg"
)

type serverContext struct {
	PackageName string
	ServerBind  string
	WwwRoot     string
	Debug       bool

	app *App
}

func AddCommonServingOptions(serverBind,
	wwwRoot, charsetDynamic, charsetStatic, indices string, debug bool) {

	optarg.Add("w", "www_root",
		"Filesystem path of the document publishing root", wwwRoot)
	optarg.Add("a", "network_address", "The IPv4 or IPv6 address to which "+
		"the generated server will bind by default", serverBind)
	optarg.Add("x", "debug", "Print debugging output", debug)
	optarg.Add("", "charset_dynamic", "Set as the charset for rendered "+
		"and negotiated resources of Content-Type text/*", charsetDynamic)
	optarg.Add("", "charset_static", "Set as the charset for static "+
		"resources of Content-Type text/*", charsetStatic)
	optarg.Add("", "indices", "A comma-separated list of filenames to "+
		"look for when a directory is requested directly; prefix "+
		"with + to extend previous configuration instead of overriding",
		indices)
}

func RunServerMain(wwwRoot, serverBind, packageName,
	charsetDynamic, charsetStatic, indices string) {

	debug := false

	AddCommonServingOptions(serverBind,
		wwwRoot, charsetDynamic, charsetStatic, indices, debug)
	for opt := range optarg.Parse() {
		switch opt.Name {
		case "network_address":
			serverBind = opt.String()
		case "www_root":
			wwwRoot = opt.String()
		case "debug":
			debug = opt.Bool()
		case "charset_dynamic":
			charsetDynamic = opt.String()
		case "charset_static":
			charsetStatic = opt.String()
		case "indices":
			indices = opt.String()
		}
	}

	wwwRoot, err := filepath.Abs(wwwRoot)
	if err != nil {
		log.Fatal(err)
	}

	SetDebug(debug)

	debugf("Declaring app for package %q", packageName)
	app := DeclareApp(packageName)
	app.Configure(serverBind, wwwRoot, charsetDynamic, charsetStatic,
		indices, debug)

	err = app.RunServer()
	if err != nil {
		log.Fatal(err)
	}
}

func newServerContext(app *App, packageName, serverBind, wwwRoot string,
	debug bool) *serverContext {

	return &serverContext{
		PackageName: packageName,
		ServerBind:  serverBind,
		WwwRoot:     wwwRoot,
		Debug:       debug,

		app: app,
	}
}

func (me *serverContext) serverQuitListener() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGQUIT)
	<-ch
	log.Println("Received SIGQUIT; exiting")
	os.Exit(0)
}

func (me *serverContext) Run() error {
	go me.serverQuitListener()

	fmt.Printf("%s-http-server serving on %q\n", me.PackageName, me.ServerBind)
	return http.ListenAndServe(me.ServerBind, nil)
}
