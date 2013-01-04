LIBRARIES = \
	github.com/meatballhat/goaspen
TARGETS = \
	$(LIBRARIES) \
	github.com/meatballhat/goaspen/goaspen-build

test: build
	go test -x -v $(LIBRARIES)

build: deps
	go install -x $(TARGETS)

deps:
	go get -n -x $(TARGETS)

clean:
	go clean -x -i $(TARGETS)

smoke: test
	$(MAKE) -C examples/smoke-test-site clean smoke

.PHONY: test build deps clean smoke
