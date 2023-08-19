# All make targets should be implemented in make/*.mk

_run:
	@$(MAKE) --warn-undefined-variables -f make/common.mk $(MAKECMDGOALS)
.PHONY: _run
$(if $(MAKECMDGOALS),$(MAKECMDGOALS): %: _run)
