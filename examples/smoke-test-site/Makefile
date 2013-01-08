GOASPEN_BUILD_OPTS := -w docroot -P goaspen_smoke_test \
	--indices +default.htm --list_directories \
	-x -s --changes_reload

smoke: docroot/Sandwich/Factory/meats\ and\ cheeses/default.htm
	GOASPEN_CONFIGURATION_SCRIPTS=./web-config.go \
		  DEBUG=1 GOPATH=./:$(GOPATH) goaspen-build $(GOASPEN_BUILD_OPTS)

clean:
	rm -vf ./bin/goaspen_smoke_test-http-server
	rm -rvf ./src/goaspen_smoke_test
	rm -vf ./docroot/.goaspen-index.json
	GOPATH=./:$(GOPATH) go clean -x -i goaspen_smoke_test

prep:
	if [ -z "$(GOASPEN_NOVPATHS)" ] ; then \
		rsync -avz ./.vpaths/ ./docroot ; \
	else \
		git clean -xfd ./docroot ; \
	fi

docroot/Sandwich/Factory/meats\ and\ cheeses/default.htm:
	mkdir -p docroot/Sandwich/Factory/meats\ and\ cheeses
	curl -s http://en.wikipedia.org/wiki/Reuben_sandwich > "$@"

.PHONY: smoke clean prep
