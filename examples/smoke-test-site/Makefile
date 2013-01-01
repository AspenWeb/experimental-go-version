GOASPEN_BUILD_OPTS := -w docroot -P goaspen_smoke_test -x -s --changes_reload

smoke:
	GOPATH=./:$(GOPATH) goaspen-build $(GOASPEN_BUILD_OPTS)

clean:
	GOPATH=./:$(GOPATH) go clean -x -i goaspen_smoke_test

.PHONY: smoke clean
