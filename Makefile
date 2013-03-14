LIBRARIES = \
	github.com/gittip/aspen-go
TARGETS = \
	$(LIBRARIES) \
	github.com/gittip/aspen-go/aspen-go-build

test: build
	go test -x -v $(GOFLAGS) $(LIBRARIES)

build: deps
	go install -x $(GOFLAGS) $(TARGETS)

deps:
	go get -n -x $(GOFLAGS) $(TARGETS)

clean:
	go clean -x -i $(TARGETS)

smoke: test
	$(MAKE) -C examples/smoke-test-site clean prep smoke

smoke-serve: test
	$(MAKE) -C examples/smoke-test-site clean prep serve

.PHONY: test build deps clean smoke smoke-serve
