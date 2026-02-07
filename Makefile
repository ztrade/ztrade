CGO := 1
PROJECTNAME := $(shell basename "$(PWD)")

build: build-ztrade

build-ztrade:
	CGO_ENABLED=$(CGO) go build  -o dist/ztrade ./

build-igo:
	CGO_ENABLED=0 go build -ldflags="-checklinkname=0" -tags ixgo -o dist/ztrade ./

.PHONY: help
all: help
help: Makefile
	@echo
	@echo "run build"
	@echo
