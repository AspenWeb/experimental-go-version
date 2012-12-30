package goaspen

import (
	"fmt"
	"net/http"
)

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
