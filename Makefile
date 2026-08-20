APP       := psfax
DIST      := dist
GO        ?= go
VERSION   ?= dev
LDFLAGS   := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := build
.PHONY: all build universal test vet fmt check clean FORCE

all: check build
build: universal

universal: $(DIST)/$(APP)_arm64 $(DIST)/$(APP)_amd64
	lipo -create -output $(DIST)/$(APP) $^
	strip $(DIST)/$(APP)
	@lipo -info $(DIST)/$(APP)

$(DIST)/$(APP)_arm64: FORCE $(wildcard *.go) go.mod .tool-versions
	@mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $@ .

$(DIST)/$(APP)_amd64: FORCE $(wildcard *.go) go.mod .tool-versions
	@mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $@ .

FORCE:

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test vet
	@test -z "$$(gofmt -l .)" || (echo "gofmt required:"; gofmt -l .; exit 1)

clean:
	rm -rf $(DIST)
