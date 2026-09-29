GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*")
RELEASE_TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
RELEASE_BINARY_NAME := ghttp
RELEASE_ARGS ?=
PUBLISH_RELEASE_ARGS ?=
RELEASE_ARTIFACT_TARGETS ?= release-artifacts container-artifacts pages-artifact
RELEASE_TOOL_DIR := $(abspath $(CURDIR)/scripts/release)
RELEASE_HELPER := $(RELEASE_TOOL_DIR)/release_helper.py
DOCKER_IMAGE ?= ghcr.io/tyemirov/ghttp
PUBLISH_PLATFORMS ?= linux/amd64,linux/arm64
PAGES_URL ?= https://ghttp.mprlab.com/
PAGES_BRANCH ?= gh-pages
PAGES_VERSION ?=
PAGES_DEPLOY_ARGS ?=

.PHONY: format check-format lint check-no-unit-tests test test-integration test-integration-coverage-gate test-pages-release build release release-artifacts container-artifacts pages-artifact publish-release publish deploy pages-deploy ci

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

test: check-no-unit-tests test-integration test-integration-coverage-gate test-pages-release

test-pages-release:
	bash tests/pages_release_pipeline_test.sh

build:
	mkdir -p bin
	go build -o bin/ghttp .

release:
	@RELEASE_HELPER="$(RELEASE_HELPER)" RELEASE_ARTIFACT_TARGETS="$(RELEASE_ARTIFACT_TARGETS)" "$(RELEASE_TOOL_DIR)/prepare_release.sh" $(RELEASE_ARGS)

release-artifacts:
	@test -n "$(RELEASE_ARTIFACT_DIR)" || { echo "error: RELEASE_ARTIFACT_DIR is required" >&2; exit 1; }
	@asset_dir="$(RELEASE_ARTIFACT_DIR)/payloads/release-assets"; \
	rm -rf "$$asset_dir/bin"; \
	mkdir -p "$$asset_dir/bin"; \
	for target in $(RELEASE_TARGETS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		extension=""; \
		if [ "$$os" = "windows" ]; then extension=".exe"; fi; \
		output_path="$$asset_dir/bin/$(RELEASE_BINARY_NAME)_$${os}_$${arch}$${extension}"; \
		echo "Building $$output_path"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags="-s -w" -o "$$output_path" ./cmd/ghttp; \
	done; \
	(cd "$$asset_dir/bin" && shasum -a 256 $(RELEASE_BINARY_NAME)_* > checksums.txt)

container-artifacts:
	@"$(RELEASE_TOOL_DIR)/prepare_container_artifact.sh" --name ghttp --image "$(DOCKER_IMAGE)" --file Dockerfile --context . --platforms "$(PUBLISH_PLATFORMS)"

pages-artifact:
	@"$(RELEASE_TOOL_DIR)/prepare_pages_artifact.sh" --source docs --domain ghttp.mprlab.com

publish-release:
	@RELEASE_HELPER="$(RELEASE_HELPER)" "$(RELEASE_TOOL_DIR)/publish_release.sh" $(PUBLISH_RELEASE_ARGS)

publish: publish-release
	@"$(RELEASE_TOOL_DIR)/publish_container_artifacts.sh"

deploy: pages-deploy

pages-deploy:
	@"$(RELEASE_TOOL_DIR)/deploy_pages_artifact.sh" --branch "$(PAGES_BRANCH)" --url "$(PAGES_URL)" $(if $(PAGES_VERSION),--version "$(PAGES_VERSION)") $(PAGES_DEPLOY_ARGS)

ci: check-format lint test
