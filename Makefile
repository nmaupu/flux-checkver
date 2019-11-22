BIN=bin
BIN_NAME=flux-checkver

all: build

fmt:
	go fmt ./...

build $(BIN)/$(BIN_NAME): $(BIN) vendor
	env CGO_ENABLED=0 go build -o $(BIN)/$(BIN_NAME)

install:
	env CGO_ENABLED=0 go install

clean:
	go clean -i
	rm -rf $(BIN) vendor

test:
	go test `go list ./... | grep -v /vendor/`

$(BIN):
	mkdir -p $(BIN)

vendor:
	export GO111MODULE=on && go mod vendor

update-vendor:
	export GO111MODULE=on && go mod vendor

.PHONY: fmt build install clean test all update-vendor
