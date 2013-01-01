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

type serverWrapper struct {
	PackageName string
	ServerBind  string
	WwwRoot     string
}

func AddCommonServingOptions(serverBind, wwwRoot string, debug bool) {
	optarg.Add("w", "www_root",
		"Filesystem path of the document publishing root", wwwRoot)
	optarg.Add("a", "network_address", "The IPv4 or IPv6 address to which "+
		"the generated server will bind by default", serverBind)
	optarg.Add("x", "debug", "Print debugging output", debug)
}

func RunServerMain(wwwRoot, serverBind, packageName string) {
	debug := false

	AddCommonServingOptions(serverBind, wwwRoot, debug)
	for opt := range optarg.Parse() {
		switch opt.Name {
		case "network_address":
			serverBind = opt.String()
		case "www_root":
			wwwRoot = opt.String()
		case "debug":
			debug = opt.Bool()
		}
	}

	wwwRoot, err := filepath.Abs(wwwRoot)
	if err != nil {
		log.Fatal(err)
	}

	SetDebug(debug)

	server := newServerWrapper(packageName, serverBind, wwwRoot)

	err = server.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func newServerWrapper(packageName, serverBind, wwwRoot string) *serverWrapper {
	return &serverWrapper{
		PackageName: packageName,
		ServerBind:  serverBind,
		WwwRoot:     wwwRoot,
	}
}

func (me *serverWrapper) serverQuitListener() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGQUIT)
	<-ch
	log.Println("Received SIGQUIT; exiting")
	os.Exit(0)
}

func (me *serverWrapper) Run() error {
	go me.serverQuitListener()

	err := expandAllHandlerFuncRegistrations(me.WwwRoot)
	if err != nil {
		return err
	}

	err = registerAllHandlerFuncs()
	if err != nil {
		return err
	}

	fmt.Printf("%s-http-server serving on %q\n", me.PackageName, me.ServerBind)
	return http.ListenAndServe(me.ServerBind, nil)
}
