APP       := psfax
VERSION   ?= 1.0.0
DIST      := dist
GO        := asdf exec go
LDFLAGS   := -s -w

.DEFAULT_GOAL := build
.PHONY: all build universal package test vet fmt check clean

all: check package
build: universal

universal: $(DIST)/$(APP)_arm64 $(DIST)/$(APP)_amd64
	lipo -create -output $(DIST)/$(APP) $^
	strip $(DIST)/$(APP)
	@lipo -info $(DIST)/$(APP)

$(DIST)/$(APP)_arm64: $(wildcard *.go) go.mod .tool-versions
	@mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $@ .

$(DIST)/$(APP)_amd64: $(wildcard *.go) go.mod .tool-versions
	@mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $@ .

package: universal
	tar -czf $(DIST)/$(APP)-macos-universal-$(VERSION).tar.gz -C $(DIST) $(APP)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test vet
	@test -z "$$(asdf exec gofmt -l .)" || (echo "gofmt required:"; asdf exec gofmt -l .; exit 1)

clean:
	rm -rf $(DIST)
