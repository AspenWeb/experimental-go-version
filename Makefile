TARGETS = \
	github.com/meatballhat/box-o-sand/gotime/smplt

test: build
	go test -x -v $(TARGETS)

build: deps
	go install -x $(TARGETS)

deps:
	go get -n -x $(TARGETS)

clean:
	go clean -x -i $(TARGETS)

.PHONY: test build clean fmt
