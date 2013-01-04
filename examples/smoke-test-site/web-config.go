package main

import (
	"fmt"
	"os"

	. "github.com/meatballhat/goaspen"
)

func configure(website *Website) *Website {
	website.Debug = true
	website.Indices = append(website.Indices, "default.htm")
	return website
}

func main() {
	fmt.Fprintf(os.Stderr, "web-config.go: DEBUG: called!\n")

	website := MustLoadWebsite()
	fmt.Fprintf(os.Stderr, "web-config.go: DEBUG: loaded website %+v\n", website)

	website = configure(website)

	fmt.Fprintf(os.Stderr, "web-config.go: DEBUG: dumping website %+v\n", website)
	MustDumpWebsite(website)
	os.Exit(0)
}
