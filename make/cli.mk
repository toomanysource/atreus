##@ Cli Development

# ================================================
# Public Commands:
# ================================================

.PHONY: wire
wire: ## Generate wire_gen code based on every wire.go under app
wire: wire.gen

# ================================================
# Private Commands:
# ================================================

.PHONY: wire.init
wire.init:
ifeq ($(shell which wire),)
	@echo "======> Installing missing wire"
	@go install github.com/google/wire/cmd/wire@v0.5.0
endif

.PHONY: wire.gen
wire.gen: wire.init
	@echo "======> Generating wire_gen code"
	@echo $(abspath $(dir $(shell find app -name wire.go))) | xargs -I{} sh -c 'wire gen {}'