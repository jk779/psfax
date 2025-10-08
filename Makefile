APP     := psfax
VERSION := 1.0.0
BUILD   := build

all: clean $(APP) package

$(APP): $(BUILD)/$(APP)_arm64 $(BUILD)/$(APP)_amd64
	lipo -create -output $(APP) $(BUILD)/$(APP)_arm64 $(BUILD)/$(APP)_amd64
	strip $(APP)
	@echo "✅ Built universal binary: $(APP)"
	@lipo -info $(APP)

$(BUILD)/$(APP)_arm64:
	mkdir -p $(BUILD)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $@ .

$(BUILD)/$(APP)_amd64:
	mkdir -p $(BUILD)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $@ .

package: $(APP)
	tar -czf $(APP)-macos-universal-$(VERSION).tar.gz $(APP)
	@echo "📦 Created release archive: $(APP)-macos-universal-$(VERSION).tar.gz"

clean:
	rm -rf $(BUILD) $(APP) $(APP)-macos-universal-*.tar.gz