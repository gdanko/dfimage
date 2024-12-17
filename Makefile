GOPATH := $(shell go env GOPATH)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
DFIMAGE_VERSION := "0.1.1"

GOOS ?= $(shell uname | tr '[:upper:]' '[:lower:]')
GOARCH ?=$(shell arch)

.PHONY: all build install

all: build install

.PHONY: mod-tidy
mod-tidy:
	go mod tidy

.PHONY: build OS ARCH
build: guard-DFIMAGE_VERSION mod-tidy clean
	@echo "================================================="
	@echo "Building dfimage"
	@echo "=================================================\n"

	@if [ ! -d "bin" ]; then \
		mkdir "bin"; \
	fi
	GOOS=${GOOS} GOARCH=${GOARCH} go build -o "bin/dfimage"
	sleep 2
	tar -czvf "dfimage_${DFIMAGE_VERSION}_${GOOS}_${GOARCH}.tgz" bin; \

.PHONY: clean
clean:
	@echo "================================================="
	@echo "Cleaning dfimage"
	@echo "=================================================\n"
	@if [ -f bin/dfimage ]; then \
		rm -f bin/dfimage; \
	fi; \

.PHONY: clean-all
clean-all: clean
	@echo "================================================="
	@echo "Cleaning tarballs"
	@echo "=================================================\n"
	@rm -f *.tgz 2>/dev/null

.PHONY: install
install:
	@echo "================================================="
	@echo "Installing dfimage in ${GOPATH}/bin"
	@echo "=================================================\n"

	go install -race

#
# General targets
#
guard-%:
	@if [ "${${*}}" = "" ]; then \
		echo "Environment variable $* not set"; \
		exit 1; \
	fi
