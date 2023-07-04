NAME := tfupdate

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
lint:
	golangci-lint run ./...

.PHONY: test
test: build
	go test ./...

.PHONY: testacc
testacc: install testacc-lock-simple

.PHONY: testacc-lock-simple
testacc-lock-simple: install
	scripts/testacc/lock.sh run simple

.PHONY: testacc-lock-debug
testacc-lock-debug: install
	scripts/testacc/lock.sh $(ARG)

.PHONY: testacc-all
testacc-all: install
	scripts/testacc/all.sh

.PHONY: check
check: lint test
