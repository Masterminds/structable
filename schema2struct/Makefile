VERSION := $(shell git describe --tags)
DIST_DIRS := find * -type d -exec

build:
	go build -o schema2struct -ldflags "-X main.version=${VERSION}" schema2struct.go

install: build
	install -d ${DESTDIR}/usr/local/bin/
	install -m 755 ./schema2struct ${DESTDIR}/usr/local/bin/schema2struct

.PHONY: build test install clean 
