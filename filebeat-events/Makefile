.PHONY: all build linux windows clean test run

BINARY_NAME=filebeat-events
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

all: build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME)

linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64

windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*

test:
	go test -v ./...

run:
	go run main.go
