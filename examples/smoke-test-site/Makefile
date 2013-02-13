smoke: docroot/Sandwich/Factory/meats\ and\ cheeses/default.htm
	./smoke-suite

serve:
	./build
	./bin/aspen_go_smoke_test-http-server

clean:
	GOPATH=$(shell pwd):$(GOPATH) go clean -x -i aspen_go_smoke_test 2>/dev/null || true
	rm -vf ./bin/aspen_go_smoke_test-http-server
	rm -rvf ./src/aspen_go_smoke_test
	rm -vf ./docroot/.aspen-go-index.json

prep:
	rsync -avz ./.vpaths/ ./docroot

docroot/Sandwich/Factory/meats\ and\ cheeses/default.htm:
	mkdir -p docroot/Sandwich/Factory/meats\ and\ cheeses
	curl -s http://en.wikipedia.org/wiki/Reuben_sandwich > "$@"

.PHONY: smoke serve clean prep
