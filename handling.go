package goaspen

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	HandlerFuncRegistrations = map[string](*HandlerFuncRegistration){}
)

type HandlerFuncRegistration struct {
	RequestPath string
	HandlerFunc func(http.ResponseWriter, *http.Request)
}

func NewHandlerFuncRegistration(requestPath string,
	handler func(http.ResponseWriter, *http.Request)) *HandlerFuncRegistration {

	if len(requestPath) < 1 {
		panic(errors.New(fmt.Sprintf("Invalid request path %q", requestPath)))
	}

	HandlerFuncRegistrations[requestPath] = &HandlerFuncRegistration{
		RequestPath: requestPath,
		HandlerFunc: handler,
	}

	return HandlerFuncRegistrations[requestPath]
}

func RegisterAllHandlerFuncs() error {
	for _, reg := range HandlerFuncRegistrations {
		http.HandleFunc(reg.RequestPath, reg.HandlerFunc)
	}

	return nil
}
