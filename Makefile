GO_MK = build/go.mk

.PHONY: setup
setup: $(GO_MK)
$(GO_MK): gen-makefile.sh
	mkdir -p build
	./gen-makefile.sh > $(GO_MK)

dist:
	mkdir -p dist

clean:
	rm -rf dist
	rm -rf build

include build/go.mk
