.PHONY: help clean godoc pkgsite reportcard test

help: ## list available targets
	@# Shamelessly stolen from Gomega's Makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

clean: ## cleans up build and testing artefacts
	rm -f coverage.html coverage.out coverage.txt

pkgsite: ## serves Go documentation on port 6060
	@PATH=$(PATH):$(shell go env GOPATH)/bin; pkgsite -http=:6060 .
	echo "navigate to: http://localhost:6060/github.com/thediveo/whalewatcher"

godoc: ## deprecated: serves godoc on port 6060 -- use "make pkgsite" instead.
	@PATH=$(PATH):$(shell go env GOPATH)/bin; godoc -http=:6060

report: ## run goreportcard on this module
	@scripts/goreportcard.sh

test: ## run unit tests
	go test -v -p=1 -race -exec sudo ./... && go test -v -p=1 -race ./...
