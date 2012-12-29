package goaspen

import (
	"log"
	"os"
)

var (
	isDebug     = false
	isDebugAddr = &isDebug
)

func init() {
	*isDebugAddr = len(os.Getenv("DEBUG")) > 0
}

func debugf(format string, v ...interface{}) {
	if isDebug {
		log.Printf("DEBUG:"+format, v...)
	}
}

func SetDebug(truth bool) {
	*isDebugAddr = truth
}
