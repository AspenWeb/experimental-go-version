package goaspen

import (
	"fmt"
	"net/http"
)

func RunServer(packageName, serverBind, siteRoot string) error {
	err := RegisterAllHandlerFuncs()
	if err != nil {
		return err
	}

	fmt.Printf("%s-server serving on %q\n", packageName, serverBind)
	http.ListenAndServe(serverBind, nil)
	return nil
}
