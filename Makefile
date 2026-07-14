IMAGE     ?= honeybadger-mcp-server
TAG       ?= hosted-test
APIGO_DIR ?= ../api-go
# Default to prod; override HONEYBADGER_URL / DOCS_URL in the environment
# (e.g. via .envrc) to point docker-run at local services.
HONEYBADGER_URL ?= https://app.honeybadger.io
DOCS_URL        ?= https://docs.honeybadger.io
MCP_NAME       ?= honeybadger-dev
MCP_PORT       ?= 9090
MCP_PUBLIC_URL ?= http://localhost:$(MCP_PORT)
MCP_URL        ?= $(MCP_PUBLIC_URL)/mcp

.PHONY: build test docker docker-local docker-run claude-mcp-add claude-mcp-remove

build:
	go build -o honeybadger-mcp-server ./cmd/honeybadger-mcp-server

test:
	go test ./...

docker-run:
	docker run --rm --network=host \
		-e HONEYBADGER_API_URL=$(HONEYBADGER_URL) \
		-e HONEYBADGER_INSTRUCTIONS_URL=$(DOCS_URL)/resources/llms/instructions \
		-e MCP_ADDRESS=:$(MCP_PORT) \
		-e MCP_PUBLIC_URL=$(MCP_PUBLIC_URL) \
		-e MCP_AUTHORIZATION_SERVER_URL=$(HONEYBADGER_URL) \
		-e LOG_LEVEL=debug \
		$(IMAGE):$(TAG) http

# Register the locally-running http host with Claude Code (uses OAuth against
# whatever MCP_AUTHORIZATION_SERVER_URL the container was started with).
claude-mcp-add:
	claude mcp add --transport http $(MCP_NAME) $(MCP_URL)

claude-mcp-remove:
	claude mcp remove $(MCP_NAME)

# Image with release dependencies, as the production pipeline builds it.
docker:
	docker build -t $(IMAGE):$(TAG) .

# Image built against the local api-go checkout (whatever branch it has
# checked out) instead of the go.mod release.
docker-local:
	docker buildx build -f Dockerfile.local --build-context apigo=$(APIGO_DIR) \
		-t $(IMAGE):$(TAG) --load .
