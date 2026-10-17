.PHONY: build clean install test run

build:
	go build -o bin/kirotoggle .

clean:
	rm -rf bin/

install: build
	mkdir -p ~/bin
	cp bin/kirotoggle ~/bin/
	chmod +x ~/bin/kirotoggle

test:
	go test ./... -v

run: build
	./bin/kirotoggle
