smoke: docroot/Sandwich/Factory/meats\ and\ cheeses/default.htm
	./smoke-suite

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
