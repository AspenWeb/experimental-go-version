smoke:
	GOPATH=./:$(GOPATH) goaspen-build -d docroot -p goaspen_smoke_test -x
	bin/goaspen_smoke_test-http-server -d docroot

.PHONY: smoke
