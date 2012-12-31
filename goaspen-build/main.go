package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/jteeuwen/go-pkg-optarg"
	"github.com/meatballhat/goaspen"
)

var (
	usageInfoTmpl = `Usage: %s [options]

By default, goaspen-build will build simplates found in the "www root" (-w)
into Go sources written to generated package (-p) in the output GOPATH base
(-o), optionally running 'go fmt' (-F).  The output GOPATH base must already
exist, or the '-m' flag may be passed to ensure it exists.
`
	usageInfo = ""
)

func init() {
	usageInfoAddr := &usageInfo
	*usageInfoAddr = fmt.Sprintf(usageInfoTmpl, path.Base(os.Args[0]))
}

func changeMonitor(q chan bool) {
	// TODO implement real monitoring with github.com/kless/go-exp/inotify
	log.Println("Simulating change in 1 second!")
	time.Sleep(1000 * time.Millisecond)
	q <- true
}

func main() {
	wwwRoot, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory! ", err)
	}

	changesReload := false
	//charsetDynamic := goaspen.DefaultCharsetDynamic
	//charsetStatic := ""
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
	goaspen.AddCommonServingOptions(genServerBind, wwwRoot, debug)
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
	// TODO
	//optarg.Add("", "changes_reload", "Changes reload.  If set to true/1, "+
	//"changes to configuration files and document root files will cause "+
	//"simplates to rebuild, then re-exec the generated server binary.",
	//changesReload)

	// TODO
	//optarg.Header("Extended Options")
	//optarg.Add("", "charset_dynamic", "Set as the charset for rendered "+
	//"and negotiated resources of Content-Type text/*", charsetDynamic)
	//optarg.Add("", "charset_static", "Set as the charset for static "+
	//"resources of Content-Type text/*", charsetStatic)

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
		case "run_server":
			value := opt.Bool()
			runServer = value
			compile = value
		}
	}

	goaspen.SetDebug(debug)

	retcode := 0

	for {
		retcode = goaspen.BuildMain(&goaspen.SiteBuilderCfg{
			RootDir:       wwwRoot,
			OutputGopath:  outPath,
			GenPackage:    genPkg,
			GenServerBind: genServerBind,
			Format:        format,
			MkOutDir:      mkOutDir,
			Compile:       compile,
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
			srvCmd.Start()

			if <-q {
				srvCmd.Process.Signal(syscall.SIGQUIT)
			}
			// TODO listen to an os.Signal channel or some such instead of waiting
			err = srvCmd.Wait()
			if _, ok := err.(*exec.ExitError); ok {
				ret <- 9
				return
			}

			ret <- 0
			return
		}(retChan, quitChan)

		if changesReload {
			go changeMonitor(quitChan)
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
