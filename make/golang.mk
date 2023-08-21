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

.PHONY: format
format: ## Format codes style with gofumpt and goimports.
format: go.format

.PHONY: clean
clean: ## clean all unused generated files.
clean: go.clean

# ================================================
# Private Commands:
# ================================================

.PHONY: go.clean
go.clean:
	@echo "======> Cleaning all unused generated files"
	@rm -rf $(OUTPUT_DIR)

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
	$(GO_FMT) -w .
	$(GOPATH)/bin/$(GO_IMPORTS) -w -local $(GO_MODULE) .
	go mod tidy