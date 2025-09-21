GO_MAKEFILE = build/go.mk

.PHONY: setup
setup: $(GO_MAKEFILE)
$(GO_MAKEFILE): gen-makefile.sh
	mkdir -p build
	./gen-makefile.sh > $(GO_MAKEFILE)

dist:
	mkdir -p dist

clean:
	rm -rf dist
	rm -rf build

include $(GO_MAKEFILE)

.PHONY: test
test:
	go test ${TEST_ARGS} ./...

