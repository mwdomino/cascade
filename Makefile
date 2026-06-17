.PHONY: build test lint run clean

build:
	go build -o cascade ./cmd/cascade

test:
	go test ./...

lint:
	go vet ./...

run: build
	./cascade

clean:
	rm -f cascade
	go clean -testcache
