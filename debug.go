package goaspen

import (
	"log"
	"os"
)

var (
	isDebug = false
)

func init() {
	*(&isDebug) = len(os.Getenv("DEBUG")) > 0
}

func debugf(format string, v ...interface{}) {
	if isDebug {
		log.Printf("DEBUG:"+format, v...)
	}
}

func SetDebug(truth bool) {
	*(&isDebug) = truth
}
