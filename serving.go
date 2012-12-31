package goaspen

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jteeuwen/go-pkg-optarg"
)

func AddCommonServingOptions(serverBind, wwwRoot string, debug bool) {
	optarg.Add("w", "www_root",
		"Filesystem path of the document publishing root", wwwRoot)
	optarg.Add("a", "network_address", "The IPv4 or IPv6 address to which "+
		"the generated server will bind by default", serverBind)
	optarg.Add("x", "debug", "Print debugging output", debug)
}

func RunServerMain(defaultRootDir, genServerBind, packageName string) {
	wwwRoot := defaultRootDir
	serverBind := genServerBind
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

	SetDebug(debug)

	go serverQuitListener()

	err := RunServer(packageName, serverBind, wwwRoot)
	if err != nil {
		log.Fatal(err)
	}
}

func serverQuitListener() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGQUIT)
	<-ch
	log.Println("Received SIGQUIT; exiting")
	os.Exit(0)
}

func RunServer(packageName, serverBind, siteRoot string) error {
	err := ExpandAllHandlerFuncRegistrations()
	if err != nil {
		return err
	}

	err = RegisterAllHandlerFuncs()
	if err != nil {
		return err
	}

	fmt.Printf("%s-server serving on %q\n", packageName, serverBind)
	return http.ListenAndServe(serverBind, nil)
}
