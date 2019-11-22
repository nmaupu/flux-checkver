BIN=bin
BIN_NAME=flux-checkver
IMAGE_NAME ?= flux-checkver
IMAGE_VERSION = 1.0.0
IMAGE_REMOTE_NAME ?= nmaupu/$(IMAGE_NAME):$(IMAGE_VERSION)

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: build
build $(BIN)/$(BIN_NAME): $(BIN) vendor
	env CGO_ENABLED=0 go build -o $(BIN)/$(BIN_NAME)
	env CGO_ENABLED=0 go build -o $(BIN)/$(BIN_NAME)

.PHONY: build-x86_64
build-x86_64 $(BIN)/$(BIN_NAME)-x86_64: $(BIN) vendor
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BIN)/$(BIN_NAME)-x86_64

.PHONY: install
install:
	env CGO_ENABLED=0 go install

.PHONY: clean
clean:
	go clean -i
	rm -rf $(BIN) vendor

.PHONY: test
test:
	go test `go list ./... | grep -v /vendor/`

$(BIN):
	mkdir -p $(BIN)

vendor:
	export GO111MODULE=on && go mod vendor

.PHONY: update-vendor
update-vendor:
	export GO111MODULE=on && go mod vendor

.PHONY: image-build
image-build: build-x86_64
	docker build -f Dockerfile.minideb -t $(IMAGE_NAME) .

image-tag: image-build
	docker tag $(IMAGE_NAME) $(IMAGE_REMOTE_NAME)

image-push: image-tag
	docker push $(IMAGE_REMOTE_NAME)
