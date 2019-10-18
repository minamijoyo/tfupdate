NAME := tfupdate

ifndef GOBIN
GOBIN := $(shell echo "$${GOPATH%%:*}/bin")
endif

GOLINT := $(GOBIN)/golint
GORELEASER := $(GOBIN)/goreleaser

$(GOLINT): ; @go install golang.org/x/lint/golint
$(GORELEASER): ; @go install github.com/goreleaser/goreleaser

.DEFAULT_GOAL := build

.PHONY: deps
deps:
	go mod download

.PHONY: build
build: deps
	go build -o bin/$(NAME)

.PHONY: install
install: deps
	go install

.PHONY: lint
lint: $(GOLINT)
	golint $$(go list ./... | grep -v /vendor/)

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: deps
	go test ./...

.PHONY: check
check: lint vet test build

.PHONY: release
release: check $(GORELEASER)
	goreleaser --rm-dist
