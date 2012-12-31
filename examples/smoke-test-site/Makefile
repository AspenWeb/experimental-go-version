smoke:
	GOPATH=./:$(GOPATH) goaspen-build -w docroot -P goaspen_smoke_test -x
	bin/goaspen_smoke_test-http-server -w docroot -x

clean:
	GOPATH=./:$(GOPATH) go clean -x -i goaspen_smoke_test

.PHONY: smoke clean
