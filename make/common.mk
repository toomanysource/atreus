# ================================================
# Common Variables:
# ================================================

SHELL := /bin/bash

COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

ifeq ($(origin ROOT_DIR),undefined)
ROOT_DIR := $(abspath $(shell cd $(COMMON_SELF_DIR)/.. && pwd -P))
endif
DATA_DIR := $(ROOT_DIR)/_data

# ================================================
# Colors: globel colors to share.
# ================================================

NO_COLOR := \033[0m
BOLD_COLOR := \n\033[1m
RED_COLOR := \033[0;31m
GREEN_COLOR := \033[0;32m
YELLOW_COLOR := \033[0;33m
BLUE_COLOR := \033[36m

# ================================================
# Includes:
# ================================================

include make/cli.mk
include make/help.mk
include make/proto.mk
include make/golang.mk
include make/docker.mk