GO_MAKEFILE = build/go.mk

.PHONY: setup
setup: $(GO_MAKEFILE) ## Generate a Makefile with all OSs and architectures targets
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
test: ## Execute all tests
	go test ${TEST_ARGS} ./...

.PHONY: help
help: ## print this help
	@grep --no-filename -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

lint:
	golangci-lint run -c .golangci.yml
