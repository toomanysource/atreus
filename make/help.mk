# ================================================
# Usage
# ================================================

## help: Show help info.
.PHONY: help
help:
	@echo -e "$(BOLD_COLOR)Usage:$(NO_COLOR)\n  make \033[36m<Target>\033[0m\n$(BOLD_COLOR)Targets:$(NO_COLOR)"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: show
show:
	@echo $(GOBIN)