package main

import (
	"fmt"
	"os"

	. "github.com/gittip/aspen-go"
)

func configure(website *Website) *Website {
	website.Debug = true
	website.Indices = append(website.Indices, "default.htm")
	return website
}

func debugf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, "web-config.go:DEBUG: "+format+"\n", v...)
}

func main() {
	debugf("called!")

	website := MustLoadWebsite()
	debugf("loaded website %+v", website)

	website = configure(website)

	debugf("dumping website %+v", website)
	MustDumpWebsite(website)
	os.Exit(0)
}
