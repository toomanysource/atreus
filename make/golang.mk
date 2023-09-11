GO_FMT := gofumpt
GO_IMPORTS := goimports

GO_MODULE := github.com/toomanysource/atreus

GOPATH := $(shell go env GOPATH | cut -d ':' -f 1)
ifeq ($(origin GOBIN), undefined)
	GOBIN := $(GOPATH)/bin
endif


##@ Golang Development

# ================================================
# Public Commands:
# ================================================

.PHONY: clean
clean: ## clean all unused generated files.
clean: go.clean

.PHONY: lint
lint: ## Analyze go syntax and styling of go source codes.
lint: go.lint

.PHONY: test
test: ## Run go unit tests
test: go.test

.PHONY: test-coverage
test-coverage: ## Run go unit tests with coverage
test-coverage: go.test.coverage

.PHONY: style
style: ## Check if go codes style is formatted and committed.
style: go.style

.PHONY: format
format: ## Format go codes style with gofumpt and goimports.
format: go.format

# ================================================
# Private Commands:
# ================================================

.PHONY: go.clean
go.clean:
	@echo "======> Cleaning all unused generated files"
	@rm -rf $(OUTPUT_DIR)

.PHONY: go.lint.verify
go.lint.verify:
ifeq ($(shell which golangci-lint),)
	@echo "======> Installing missing golangci-lint"
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin
endif

.PHONY: go.lint
go.lint: go.lint.verify
	@echo "======> Running golangci-lint to analyze source codes"
	@golangci-lint run -v

.PHONY: go.test
go.test:
	@echo "======> Running unit tests in app"
	@go test -count=1 -timeout=10m -short -v `go list ./app/...`

.PHONY: go.test.coverage
go.test.coverage:
	@echo "======> Running unit tests with coverage in app"
	@go test -race -v -coverprofile=coverage.out ./app/...

.PHONY: go.style
go.style:
	@echo "======> Running go style check"
	@$(MAKE) format && \
		git status && \
		[[ -z `git status -s` ]] || \
		(echo -e "\n${RED_COLOR}Error: there are uncommitted changes after formatting go codes.\n${GREEN_COLOR}You should run 'make format' then use git to commit all those changes.${NO_COLOR}" && exit 1)

.PHONY: go.format.verify
go.format.verify:
ifneq ($(shell $(GO_FMT) -version | awk '{print $$1}'),v0.5.0)
	@echo "======> Installing missing gofumpt"
	@go install mvdan.cc/gofumpt@v0.5.0
endif
ifeq ($(shell which goimports),)
	@echo "======> Installing missing goimports"
	@go install golang.org/x/tools/cmd/goimports@v0.12.0
endif

.PHONY: go.format
go.format: go.format.verify
	@echo "======> Formatting go codes"
	@find $(ROOT_DIR) -path $(DATA_DIR) -prune -false -o -name '*.go' -not -name '*.pb.*' | xargs -I{} sh -c '$(GO_FMT) -w {}'
	@find $(ROOT_DIR) -path $(DATA_DIR) -prune -false -o -name '*.go' | xargs -I{} sh -c '$(GOPATH)/bin/$(GO_IMPORTS) -w -local $(GO_MODULE) {}'
	@go mod tidy