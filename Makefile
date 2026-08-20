APP       := psfax
DIST      := dist
GO        ?= go
LDFLAGS   := -s -w

.DEFAULT_GOAL := build
.PHONY: all build universal test vet fmt check clean

all: check build
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
