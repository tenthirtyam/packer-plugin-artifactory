# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

NAME=artifactory
BINARY=packer-plugin-${NAME}
PLUGIN_FQN="$(shell grep -E '^module' <go.mod | sed -E 's/module *//')"

COUNT?=1
TEST?=$(shell go list ./...)
HASHICORP_PACKER_PLUGIN_SDK_VERSION?=$(shell go list -m github.com/hashicorp/packer-plugin-sdk | cut -d " " -f2)

VENV ?= .venv
BOOTSTRAP_PYTHON ?= $(shell \
	if command -v python3.12 >/dev/null 2>&1; then \
		command -v python3.12; \
	elif command -v python3.13 >/dev/null 2>&1; then \
		command -v python3.13; \
	elif command -v python3.11 >/dev/null 2>&1; then \
		command -v python3.11; \
	else \
		command -v python3; \
	fi)
PYTHON := $(VENV)/bin/python
PIP := $(PYTHON) -m pip
DOCS := $(PYTHON) -m properdocs
DOCS_PORT ?= 8000
DOCS_CONFIG := docs-site/properdocs.yml
DOCS_REQUIREMENTS := docs-site/requirements.txt
DOCS_STAMP := $(VENV)/.docs-installed
DOCS_SCRIPTS := docs-site/scripts

.PHONY: all build clean clean-build clean-cache clean-docs dev format generate help \
	install-docs install-packer-sdc plugin-check setup-packer-env \
	docs docs-serve docs-serve-mike docs-serve-mike-only docs-backfill docs-deploy \
	test test-acceptance test-integration test-integration-clean test-integration-down \
	test-integration-environment test-unit

help:
	@echo "Testing:"
	@echo "  make test                            Run unit and integration tests."
	@echo "  make test-unit                       Run unit tests."
	@echo "  make test-acceptance                 Run acceptance tests."
	@echo "  make test-integration                Run integration tests (local Docker by default)."
	@echo "  make test-integration-clean          Stop and remove integration Docker volumes."
	@echo "  make test-integration-down           Stop integration Docker stack."
	@echo "  make test-integration-environment    Print how to set Packer example credentials."
	@echo ""
	@echo "Integration targets:"
	@echo "  Local Docker:  make test-integration"
	@echo "  Fresh volumes: INTEGRATION_FRESH=1 make test-integration"
	@echo "  Teardown:      TEARDOWN_AFTER=1 make test-integration"
	@echo "  External:      USE_DOCKER=0 ARTIFACTORY_URL=... ARTIFACTORY_USERNAME=... \\"
	@echo "                   ARTIFACTORY_PASSWORD=... ARTIFACTORY_REPOSITORY=... \\"
	@echo "                   make test-integration"
	@echo ""
	@echo "Documentation:"
	@echo "  make install-docs                    Install documentation dependencies."
	@echo "  make docs                            Build the documentation site."
	@echo "  make docs-serve                      Serve the documentation site locally."
	@echo "  make docs-serve-mike                 Build and serve a multi-version mike preview."
	@echo "  make docs-serve-mike-only            Serve an existing mike preview branch."
	@echo "  make docs-backfill                    Publish one or more versions to gh-pages."
	@echo "  make docs-deploy VERSION=x.y.z       Deploy a single version (add UPDATE_LATEST=1)."

all: clean-build generate build test-unit

build:
	@go build -o ${BINARY}

clean: clean-build clean-cache clean-docs
	go clean -modcache

clean-build:
	rm -f ${BINARY}
	rm -rf .docs/
	rm -rf docs-partials/
	rm -f *.mdx

clean-cache:
	go clean -testcache
	go clean -cache

clean-docs:
	rm -rf site
	rm -f $(DOCS_STAMP)

$(PYTHON):
	$(BOOTSTRAP_PYTHON) -m venv $(VENV)
	$(PIP) install --upgrade pip

install-docs: $(DOCS_STAMP)

$(DOCS_STAMP): $(PYTHON) $(DOCS_REQUIREMENTS)
	@if ! $(PIP) --version >/dev/null 2>&1; then \
		echo "Rebuilding broken virtual environment at $(VENV)"; \
		rm -rf $(VENV); \
		$(MAKE) $(PYTHON); \
	fi
	$(PIP) install -r $(DOCS_REQUIREMENTS)
	@touch $(DOCS_STAMP)

docs: install-docs
	$(DOCS) build -f $(DOCS_CONFIG)

docs-serve: install-docs
	@pids="$$(lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN 2>/dev/null || true)"; \
	for pid in $$pids; do \
		echo "Stopping existing server on port $(DOCS_PORT): $$pid"; \
		kill "$$pid" 2>/dev/null || true; \
	done; \
	for attempt in 1 2 3 4 5 6 7 8 9 10; do \
		if ! lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN >/dev/null 2>&1; then \
			break; \
		fi; \
		sleep 1; \
	done; \
	if lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN >/dev/null 2>&1; then \
		echo "Force stopping server on port $(DOCS_PORT)"; \
		lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN | xargs kill -9 2>/dev/null || true; \
	fi
	$(DOCS) serve -f $(DOCS_CONFIG) --open --livereload -a 127.0.0.1:$(DOCS_PORT) -w ./

docs-serve-mike: install-docs
	@$(DOCS_SCRIPTS)/mike-preview.sh $(if $(VERSIONS),$(VERSIONS),) $(MIKE_PREVIEW_ARGS)

docs-serve-mike-only: install-docs
	@$(DOCS_SCRIPTS)/mike-preview.sh --serve-only

docs-backfill: install-docs
	@$(DOCS_SCRIPTS)/mike-backfill.sh $(if $(VERSIONS),$(VERSIONS),)

docs-deploy: install-docs
	@test -n "$(VERSION)" || (echo "Set VERSION=x.y.z (without v prefix)" && exit 1)
	@$(DOCS_SCRIPTS)/mike-deploy.sh $(VERSION) $(if $(filter 1 true yes,$(UPDATE_LATEST)),--update-latest,)

dev:
	@go build -ldflags="-X '${PLUGIN_FQN}/version.channel='" -o '${BINARY}'
	packer plugins install --path ${BINARY} "$(shell echo "${PLUGIN_FQN}" | sed 's/packer-plugin-//')"

format:
	@gofmt -w .

generate: install-packer-sdc
	@go generate ./...
	@rm -rf .docs
	@packer-sdc renderdocs -src "docs" -partials docs-partials/ -dst ".docs/"
	@./scripts/generate-docs.sh "." ".docs" ".web-docs" "hashicorp"
	@rm -r ".docs"

install-packer-sdc:
	@go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@${HASHICORP_PACKER_PLUGIN_SDK_VERSION}

plugin-check: install-packer-sdc build
	@packer-sdc plugin-check ${BINARY}

test: test-unit test-integration

test-acceptance: dev
	@PACKER_ACC=1 go test -count $(COUNT) -v $(TEST) -timeout=120m

test-integration: build
	@./scripts/test-integration.sh

test-integration-clean: test-integration-down
	@docker compose -f test/docker-compose.yaml down -v --remove-orphans

test-integration-down:
	@docker compose -f test/docker-compose.yaml down --remove-orphans

test-integration-environment setup-packer-env:
	@echo "Configure Artifactory credentials for Packer examples:"
	@echo "  source scripts/setup-packer-env.sh"

test-unit:
	@go test -race -count $(COUNT) $(TEST) -timeout=3m
