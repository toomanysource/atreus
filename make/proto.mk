API_DIR := $(ROOT_DIR)/api
APP_DIR := $(ROOT_DIR)/app
THIRD_PARTY_DIR := $(ROOT_DIR)/third_party

##@ Proto Development

# ================================================
# Public Commands:
# ================================================

.PHONY: proto
proto: ## Generate code based on the proto files under api and app.
proto: proto.gen

# ================================================
# Private Commands:
# ================================================

.PHONY: proto.gen
proto.gen:
ifeq ($(shell docker image ls 'atreus/protoc:latest' | grep 'atreus/protoc'),)
	@echo "======> Building missing docker image"
	docker build --network host -t atreus/protoc docker/proto
endif
	@echo "======> Generating pb.go files"
	docker run --rm \
		-v $(API_DIR):/pb/proto/api \
		-v $(APP_DIR):/pb/proto/app \
		-v $(THIRD_PARTY_DIR):/pb/proto/third_party \
		atreus/protoc
	$(MAKE) format
