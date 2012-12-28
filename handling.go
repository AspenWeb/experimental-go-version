package goaspen

import (
	"net/http"
)

var (
	handlerFuncRegistrations map[string]func(http.ResponseWriter, *http.Request)
)

func HandlerFuncRegistration(requestPath string,
	handler func(http.ResponseWriter, *http.Request)) error {

	handlerFuncRegistrations[requestPath] = handler
	return nil
}
