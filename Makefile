.PHONY: build clean install test run

build:
	go build -o bin/kiront .

clean:
	rm -rf bin/

install: build
	mkdir -p ~/bin
	cp bin/kiront ~/bin/
	chmod +x ~/bin/kiront

test:
	go test ./... -v

run: build
	./bin/kiront
