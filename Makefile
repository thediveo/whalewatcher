.PHONY: help chores clean coverage pkgsite report test vuln

help: ## list available targets
	@# Shamelessly stolen from Gomega's Makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

clean: ## cleans up build and testing artefacts
	rm -f coverage.html coverage.out coverage.txt
	sudo rm -f coverage-root.out

coverage: ## gathers coverage and updates README badge
	@scripts/cov.sh

pkgsite: ## serves Go documentation on port 6060
	@echo "navigate to: http://localhost:6060/github.com/thediveo/whalewatcher"
	@scripts/pkgsite.sh

report: ## run goreportcard on this module
	@scripts/goreportcard.sh

test: ## run unit tests
	go test -v -p=1 -race -exec sudo ./...
	go test -v -p=1 -race ./...

vuln: ## runs govulncheck
	@scripts/vuln.sh

grype: ## runs grype vulnerabilities scanner
	@scripts/grypes.sh

chores: ## updates Go binaries and NPM helper packages if necessary
	@scripts/chores.sh
