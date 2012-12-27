package goaspen

import (
	"log"
	"os"
)

var (
	isDebug     bool = false
	isDebugAddr      = &isDebug
)

func init() {
	*isDebugAddr = len(os.Getenv("DEBUG")) > 0
}

func debugf(format string, v ...interface{}) {
	if isDebug {
		log.Printf("DEBUG:"+format, v...)
	}
}
