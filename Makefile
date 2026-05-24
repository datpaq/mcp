.PHONY: build test lint install clean docker

build:
	go build -o bin/datpaq-mcp-http ./cmd/datpaq-mcp-http

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/datpaq-mcp-http

clean:
	rm -rf bin/

docker:
	docker build -t datpaq-mcp-http -f cmd/datpaq-mcp-http/Dockerfile .
