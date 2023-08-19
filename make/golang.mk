GOPATH := $(shell go env GOPATH | cut -d ':' -f 1)
ifeq ($(origin GOBIN), undefined)
	GOBIN := $(GOPATH)/bin
endif

##@ Golang Development