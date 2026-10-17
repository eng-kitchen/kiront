BINARY  = kiront
BUNDLE  = Kiront.app
INSTALL = $(HOME)/Applications

.PHONY: build clean install test run

build:
	go build -o $(BINARY) .
	mkdir -p $(BUNDLE)/Contents/MacOS
	mkdir -p $(BUNDLE)/Contents/Resources
	cp $(BINARY) $(BUNDLE)/Contents/MacOS/
	cp Info.plist $(BUNDLE)/Contents/
	cp -r gateway $(BUNDLE)/Contents/Resources/
	printf 'APPL????' > $(BUNDLE)/Contents/PkgInfo
	@echo "Built $(BUNDLE)"

# Install to ~/Applications, bypass Gatekeeper, and launch
install: build
	codesign --sign - --force --deep $(BUNDLE)
	mkdir -p $(INSTALL)
	rm -rf $(INSTALL)/$(BUNDLE)
	cp -r $(BUNDLE) $(INSTALL)/
	xattr -dr com.apple.quarantine $(INSTALL)/$(BUNDLE) 2>/dev/null || true
	open $(INSTALL)/$(BUNDLE)
	@echo "Installed and launched $(INSTALL)/$(BUNDLE)"

clean:
	rm -rf $(BINARY) $(BUNDLE) bin/

test:
	go test ./... -v

run: build
	./$(BUNDLE)/Contents/MacOS/$(BINARY)
