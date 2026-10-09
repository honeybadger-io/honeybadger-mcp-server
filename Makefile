IMAGE     ?= honeybadger-mcp-server
TAG       ?= hosted-test
APIGO_DIR ?= ../api-go
# Default to prod; override HONEYBADGER_URL / DOCS_URL in the environment
# (e.g. via .envrc) to point docker-run at local services.
HONEYBADGER_URL ?= https://app.honeybadger.io
DOCS_URL        ?= https://docs.honeybadger.io
MCP_NAME       ?= honeybadger-dev
# user scope makes the server visible to Claude Code in every directory.
MCP_SCOPE      ?= user
MCP_PORT       ?= 9090
MCP_PUBLIC_URL ?= http://localhost:$(MCP_PORT)
MCP_URL        ?= $(MCP_PUBLIC_URL)/mcp

.PHONY: build test verify-module verify-module-local docker docker-local docker-run claude-mcp-add claude-mcp-remove

build:
	go build -o honeybadger-mcp-server ./cmd/honeybadger-mcp-server

test:
	go test ./...

# Build the way Docker does: no workspace, no go.mod writes.
#
# ../go.work covers this module and api-go, so a local build resolves api-go's
# dependencies through the workspace and never records them here. Docker copies
# the two module directories and no go.work, so it falls back to module mode and
# fails on go.sum entries that were never added. This catches that on the host,
# where the error is one line instead of a failed image build.
#
# The fix when it fails is GOWORK=off go mod tidy.
verify-module:
	GOWORK=off go build -mod=readonly ./...

# A single container needs no shared secret, so an unset MCP_CONFIRM_SECRET
# gets a fresh random one per run; a known default would let any caller forge
# delete confirmations. Passed by name to keep the value off the command line.
docker-run:
	MCP_CONFIRM_SECRET="$${MCP_CONFIRM_SECRET:-$$(openssl rand -hex 32)}" \
	docker run --rm --network=host \
		-e HONEYBADGER_API_URL=$(HONEYBADGER_URL) \
		-e HONEYBADGER_INSTRUCTIONS_URL=$(DOCS_URL)/resources/llms/instructions \
		-e MCP_ADDRESS=:$(MCP_PORT) \
		-e MCP_PUBLIC_URL=$(MCP_PUBLIC_URL) \
		-e MCP_AUTHORIZATION_SERVER_URL=$(HONEYBADGER_URL) \
		-e MCP_CONFIRM_SECRET \
		-e LOG_LEVEL=debug \
		$(IMAGE):$(TAG) http

# Register the locally-running http host with Claude Code (uses OAuth against
# whatever MCP_AUTHORIZATION_SERVER_URL the container was started with).
claude-mcp-add:
	claude mcp add --scope $(MCP_SCOPE) --transport http $(MCP_NAME) $(MCP_URL)

claude-mcp-remove:
	claude mcp remove --scope $(MCP_SCOPE) $(MCP_NAME)

# Image with release dependencies, as the production pipeline builds it.
docker:
	docker build -t $(IMAGE):$(TAG) .

# verify-module for docker-local: build the way Dockerfile.local does, with
# api-go replaced by the local checkout, so an api-go change not yet pinned in
# go.mod doesn't fail the check. The replace goes into a throwaway copy of
# go.mod, never the real one.
verify-module-local:
	@tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	cp go.mod "$$tmp/local.mod" && cp go.sum "$$tmp/local.sum" && \
	GOWORK=off go mod edit -modfile="$$tmp/local.mod" -replace github.com/honeybadger-io/api-go=$(abspath $(APIGO_DIR)) && \
	GOWORK=off go build -modfile="$$tmp/local.mod" -mod=readonly ./...

# Image built against the local api-go checkout (whatever branch it has
# checked out) instead of the go.mod release.
docker-local: verify-module-local
	docker buildx build -f Dockerfile.local --build-context apigo=$(APIGO_DIR) \
		-t $(IMAGE):$(TAG) --load .
