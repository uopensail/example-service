CURDIR:=$(shell pwd)
OS = $(shell go env GOOS)
ARCH = $(shell go env GOARCH)
.PHONY: build clean run

GITCOMMITHASH := $(shell git rev-parse --short=7 HEAD)
GITBRANCHNAME := $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)

GOLDFLAGS += -X handler.__GITCOMMITINFO__=$(GITCOMMITHASH).${GITBRANCHNAME}
GOFLAGS = -ldflags "$(GOLDFLAGS)"

PUBLISHDIR=${CURDIR}/dist
PROJECT_NAME=example-service

all: dev

clean:
	rm -rf ${PUBLISHDIR}/

build: clean
	go build -o ${PUBLISHDIR}/${PROJECT_NAME} $(GOFLAGS)

build-docker-image: local
	mv dist/example-service dist/main
	docker build \
		--build-arg GIT_HASH=$(GITCOMMITHASH) \
		--build-arg GIT_TAG=$(GITBRANCHNAME) \
		--build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
		-t kungjim/${PROJECT_NAME}:${GITBRANCHNAME}-${GITCOMMITHASH} \
		-t kungjim/${PROJECT_NAME}:latest \
		.

push-docker-image: build-docker-image
	docker push kungjim/${PROJECT_NAME}:${GITBRANCHNAME}-${GITCOMMITHASH}
	docker push kungjim/${PROJECT_NAME}:latest

local: build
	mkdir -pv ${PUBLISHDIR}/conf/
	cp -aRf conf/$@/* ${PUBLISHDIR}/conf/