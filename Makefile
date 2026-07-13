IMAGE     ?= honeybadger-mcp-server
TAG       ?= hosted-test
APIGO_DIR ?= ../api-go

.PHONY: build test docker docker-local

build:
	go build -o honeybadger-mcp-server ./cmd/honeybadger-mcp-server

test:
	go test ./...

# Image with release dependencies, as the production pipeline builds it.
docker:
	docker build -t $(IMAGE):$(TAG) .

# Image built against the local api-go checkout (whatever branch it has
# checked out) instead of the go.mod release.
docker-local:
	docker buildx build -f Dockerfile.local --build-context apigo=$(APIGO_DIR) \
		-t $(IMAGE):$(TAG) --load .
