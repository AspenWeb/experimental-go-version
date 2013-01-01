/*
Main entry point for goaspen.

This executable's main responsibility is to parse command-line arguments and
construct a goaspen.SiteBuilderCfg which is passed to goaspen.BuildMain.

Something like a development cycle is supported via the `changes_reload` flag,
which implies both the `run_server` (-s) and `compile` (-C) flags:

    goaspen-build -w ./mysite/docroot -P mysite --changes_reload

*/
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jteeuwen/go-pkg-optarg"
	"github.com/kless/go-exp/inotify"
	"github.com/meatballhat/goaspen"
)

var (
	usageInfoTmpl = `Usage: %s [options]

By default, goaspen-build will build simplates found in the "www root" (-w)
into Go sources written to generated package (-p) in the output GOPATH base
(-o), optionally running 'go fmt' (-F).  The output GOPATH base must already
exist, or the '-m' flag may be passed to ensure it exists.
`
	usageInfo    = ""
	changeEvents = inotify.IN_CREATE | inotify.IN_DELETE | inotify.IN_MODIFY | inotify.IN_MOVE
)

func init() {
	usageInfoAddr := &usageInfo
	*usageInfoAddr = fmt.Sprintf(usageInfoTmpl, path.Base(os.Args[0]))
}

func changeMonitor(wwwRoot string, q chan bool) error {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		return err
	}

	defer watcher.Close()

	err = filepath.Walk(wwwRoot,
		func(pathEntry string, info os.FileInfo, err error) error {
			return watcher.Watch(pathEntry)
		})

	if err != nil {
		return err
	}

	for {
		select {
		case ev := <-watcher.Event:
			if ev.Mask&changeEvents != 0 {
				log.Println("Got change event:", ev)
				q <- true
				return nil
			}
		case err := <-watcher.Error:
			log.Panicln("error:", err)
		}
	}

	return nil
}

func main() {
	wwwRoot, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory! ", err)
	}

	changesReload := false
	charsetDynamic := goaspen.DefaultCharsetDynamic
	charsetStatic := goaspen.DefaultCharsetStatic
	compile := true
	//configFiles := []string{}
	debug := false
	format := true
	genPkg := goaspen.DefaultGenPackage
	genServerBind := ":9182"
	//loggingThreshold := 0
	mkOutDir := false
	outPath := goaspen.DefaultOutputGopath
	runServer := false

	optarg.UsageInfo = usageInfo

	optarg.Add("h", "help", "Show this help message and exit", false)

	optarg.Header("Serving Options")
	goaspen.AddCommonServingOptions(genServerBind,
		wwwRoot, charsetDynamic, charsetStatic, debug)
	optarg.Add("s", "run_server",
		"Start server once compiled (implies `-C`)", runServer)
	// TODO
	//optarg.Header("General Configuration Options")
	//optarg.Add("f", "configuration_files", "Comma-separated list of paths "+
	//"to configuration files in Go syntax that accept config JSON on "+
	//"stdin and write config JSON to stdout.", configFiles)
	// TODO
	//optarg.Add("l", "logging_threshold", "a small integer; 1 will suppress "+
	//"most of goaspen's internal logging, 2 will suppress all it",
	//loggingThreshold)
	optarg.Header("Source Generation & Compiling Options")
	optarg.Add("P", "package_name", "Generated source package name", genPkg)
	optarg.Add("o", "output_path",
		"Output GOPATH base for generated sources", outPath)
	optarg.Add("F", "format", "Format generated sources", format)
	optarg.Add("m", "make_outdir",
		"Make output GOPATH base if not exists", mkOutDir)
	optarg.Add("C", "compile", "Compile generated sources", compile)
	optarg.Add("", "changes_reload", "Changes reload.  If set to true/1, "+
		"changes to configuration files and document root files will cause "+
		"simplates to rebuild, then re-exec the generated server binary "+
		"(implies '--compile' and '--run_server').",
		changesReload)

	for opt := range optarg.Parse() {
		switch opt.Name {
		case "help":
			optarg.Usage()
			os.Exit(2)
		case "package_name":
			genPkg = opt.String()
		case "www_root":
			wwwRoot = opt.String()
		case "output_path":
			outPath = opt.String()
		case "format":
			format = opt.Bool()
		case "make_outdir":
			mkOutDir = opt.Bool()
		case "debug":
			debug = opt.Bool()
		case "changes_reload":
			value := opt.Bool()
			runServer = value
			compile = value
			changesReload = value
		case "run_server":
			value := opt.Bool()
			runServer = value
			compile = value
		case "charset_dynamic":
			charsetDynamic = opt.String()
		case "charset_static":
			charsetStatic = opt.String()
		}
	}

	goaspen.SetDebug(debug)

	retcode := 0

	for {
		retcode = goaspen.BuildMain(&goaspen.SiteBuilderCfg{
			WwwRoot:       wwwRoot,
			OutputGopath:  outPath,
			GenPackage:    genPkg,
			GenServerBind: genServerBind,
			Format:        format,
			MkOutDir:      mkOutDir,
			Compile:       compile,

			CharsetDynamic: charsetDynamic,
			CharsetStatic:  charsetStatic,
		})

		if !runServer {
			break
		}

		retChan := make(chan int)
		quitChan := make(chan bool)

		go func(ret chan int, q chan bool) {
			httpExe := path.Join(outPath, "bin", genPkg+"-http-server")
			srvCmd := exec.Command(httpExe,
				"-w", wwwRoot, "-a", genServerBind, "-x", fmt.Sprintf("%v", debug))
			srvCmd.Stdout = os.Stdout
			srvCmd.Stderr = os.Stderr

			cmdErr := make(chan error)

			go func() {
				cmdErr <- srvCmd.Run()
			}()

			defer func() {
				time.Sleep(3000 * time.Millisecond)
				srvCmd.Process.Kill()
			}()

			for {
				keepLooping := true
				select {
				case <-q:
					srvCmd.Process.Signal(syscall.SIGQUIT)
					keepLooping = false
					break
				case err := <-cmdErr:
					if _, ok := err.(*exec.ExitError); ok {
						ret <- 9
						return
					}
					keepLooping = false
					break
				default:
					time.Sleep(1000 * time.Millisecond)
				}

				if !keepLooping {
					break
				}
			}

			ret <- 0
			return
		}(retChan, quitChan)

		if changesReload {
			go changeMonitor(wwwRoot, quitChan)
		} else {
			quitChan <- false
		}

		retcode = <-retChan
		if retcode != 0 {
			break
		}
	}

	os.Exit(retcode)
}
