CGO := 1
PROJECTNAME := $(shell basename "$(PWD)")

build: build-ztrade copy-files

build-ztrade:
	CGO_ENABLED=$(CGO) go build  -o dist/ztrade ./
copy-files:
	cp -r files/report dist/
	cp -r configs dist/
	cp -r files/tmpl dist/

.PHONY: help
all: help
help: Makefile
	@echo
	@echo "run build"
	@echo
