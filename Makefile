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

	@if [ ! -d "${GOOS}" ]; then \
		mkdir "${GOOS}"; \
	fi
	GOOS=${GOOS} GOARCH=${GOARCH} go build -o "${GOOS}/dfimage"
	sleep 2
	tar -C "${GOOS}" -czvf "dfimage_${DFIMAGE_VERSION}_${GOOS}_${GOARCH}.tgz" dfimage; \

.PHONY: clean
clean:
	@echo "================================================="
	@echo "Cleaning dfimage"
	@echo "=================================================\n"
	@for OS in darwin linux; do \
		if [ -f $${OS}/dfimage ]; then \
			rm -f $${OS}/dfimage; \
		fi; \
	done

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
