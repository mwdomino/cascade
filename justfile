default:
    @just --list

build:
    go build -ldflags "-X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o cascade ./cmd/cascade

test:
    go test ./...

lint:
    go vet ./...

run: build
    ./cascade

clean:
    rm -f cascade
    go clean -testcache
