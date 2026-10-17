BINARY  = kiront
BUNDLE  = Kiront.app
INSTALL = $(HOME)/Applications

.PHONY: build clean install uninstall test run smoke

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

# Kill the running app and remove it from ~/Applications
uninstall:
	pkill -KILL -x $(BINARY) 2>/dev/null || true
	rm -rf $(INSTALL)/$(BUNDLE)
	@echo "Uninstalled $(INSTALL)/$(BUNDLE)"

# Full lifecycle: install → verify running → wait → verify stable → uninstall → verify gone
smoke: install
	@echo "--- waiting for process to start ---"
	@sleep 2
	@pgrep -x $(BINARY) > /dev/null \
		&& echo "PASS: process is running" \
		|| (echo "FAIL: process did not start" && exit 1)
	@echo "--- running for 5 s (crash check) ---"
	@sleep 5
	@pgrep -x $(BINARY) > /dev/null \
		&& echo "PASS: still running, no crash" \
		|| (echo "FAIL: process crashed" && exit 1)
	$(MAKE) uninstall
	@sleep 1
	@pgrep -x $(BINARY) > /dev/null \
		&& echo "FAIL: process still running after uninstall" && exit 1 \
		|| echo "PASS: process stopped cleanly"
	@echo "--- smoke test passed ---"

clean:
	rm -rf $(BINARY) $(BUNDLE) bin/

test:
	go test ./... -v

run: build
	./$(BUNDLE)/Contents/MacOS/$(BINARY)
