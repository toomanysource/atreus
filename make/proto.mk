ifeq ($(origin PROTOC_DIR),undefined)
PROTOC_DIR := $(OUTPUT_DIR)/protoc
$(shell mkdir -p $(PROTOC_DIR))
endif

##@ Proto Development

# ================================================
# Public Commands:
# ================================================

.PHONY: proto
proto: ## Generate code based on the proto files under api and app.
proto: proto.gen.all

.PHONY: proto-api
proto-api: ## Generate code based on the proto files under api.
proto-api: proto.gen.api

.PHONY: proto-app
proto-app: ## Generate code based on the proto files under app.
proto-app: proto.gen.app

# ================================================
# Private Commands:
# ================================================

.PHONY: proto.gen.init
proto.gen.init:
ifneq ($(shell protoc --version),libprotoc 24.0)
	@echo "======> Installing specified protoc"
	@curl -fsSL \
    		"https://github.com/protocolbuffers/protobuf/releases/download/v24.0/protoc-24.0-$$(uname -s)-$$(uname -m).zip" \
    		-o "$(PROTOC_DIR)/protoc"
	@unzip -q -o "$(PROTOC_DIR)/protoc" -d "$(PROTOC_DIR)"
	@cp "$(PROTOC_DIR)/bin/protoc" "$(GOBIN)"
	@chmod 775 "$(GOBIN)/protoc"
endif
ifneq ($(shell protoc-gen-go --version),protoc-gen-go v1.31.0)
	@echo "======> Installing specified protoc-gen-go"
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
endif
ifneq ($(shell protoc-gen-go-grpc --version),protoc-gen-go-grpc 1.3.0)
	@echo "======> Installing specified protoc-gen-go"
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
endif
ifneq ($(shell protoc-gen-go-http --version),protoc-gen-go-http v2.7.0)
	@echo "======> Installing specified protoc-gen-go-http"
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.0.0-20230808051727-7888107c4b4f
endif


.PHONY: proto.gen.api
proto.gen.api: proto.gen.init
	@echo "======> Generating pb.go files under api"
	@find api -name *.proto | xargs -I{} sh -c 'protoc --proto_path=./api \
                              	   					  	--proto_path=./third_party \
                               						   	--go_out=paths=source_relative:./api \
                               	   					   	--go-http_out=paths=source_relative:./api \
                               	   					  	--go-grpc_out=paths=source_relative:./api \
                               	    				   	{}'

.PHONY: proto.gen.app
proto.gen.app: proto.gen.init
	@echo "======> Generating pb.go files under app"
	@find app -name *.proto | xargs -I{} sh -c 'protoc --proto_path=./app \
                                  	   					--proto_path=./third_party \
                                   						--go_out=paths=source_relative:./app \
                                   	    				{}'

.PHONY: proto.gen.all
proto.gen.all: proto.gen.init proto.gen.api proto.gen.app