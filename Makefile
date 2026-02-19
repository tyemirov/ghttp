GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*")
RELEASE_TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
RELEASE_DIRECTORY := dist
RELEASE_BINARY_NAME := ghttp

.PHONY: format check-format lint check-no-unit-tests test test-integration test-integration-coverage-gate build release ci

format:
	gofmt -w $(GO_SOURCES)

check-format:
	@formatted_files="$$(gofmt -l $(GO_SOURCES))"; \
	if [ -n "$$formatted_files" ]; then \
		echo "Go files require formatting:"; \
		echo "$$formatted_files"; \
		exit 1; \
	fi

lint:
	go vet ./...

test-integration:
	go test ./tests/integration -count=1

test-integration-coverage-gate:
	go test ./tests/integration -run 'Test(BrowseModeBrowseHandlerCoverageGate|GlobalIntegrationCoverageGate)' -count=1

check-no-unit-tests:
	@unexpected_tests="$$(find . -name '*_test.go' -type f | grep -v '^./tests/integration/' || true)"; \
	if [ -n "$$unexpected_tests" ]; then \
		echo "Non-integration test files are not allowed:"; \
		echo "$$unexpected_tests"; \
		exit 1; \
	fi

test: check-no-unit-tests test-integration test-integration-coverage-gate

build:
	mkdir -p bin
	go build -o bin/ghttp .

release:
	rm -rf $(RELEASE_DIRECTORY)
	mkdir -p $(RELEASE_DIRECTORY)
	for target in $(RELEASE_TARGETS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		output_path=$(RELEASE_DIRECTORY)/$(RELEASE_BINARY_NAME)-$$os-$$arch; \
		echo "Building $$output_path"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -o $$output_path .; \
	done

ci: check-format lint test
