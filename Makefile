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

.PHONY: test build clean fmt
