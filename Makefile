.DEFAULT_GOAL := build

PREFIX = /usr/local

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build -ldflags="-s -w" -trimpath tv.go

clean:
	rm -f tv

install: build
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp tv $(DESTDIR)$(PREFIX)/bin/tv
	chmod 755 $(DESTDIR)$(PREFIX)/bin/tv

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/tv

.PHONY: fmt vet build clean install uninstall
