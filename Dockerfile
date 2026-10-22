FROM golang:1.26.10-alpine AS build
WORKDIR /build
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o /bin/honeybadger-mcp-server ./cmd/honeybadger-mcp-server

FROM alpine:3
# 20261020-r0 drops emSign ECC Root CA - G3 (eMudhra distrust). The floor
# fails the build rather than shipping an older bundle from a stale base.
RUN apk add --no-cache 'ca-certificates-bundle>=20261020-r0'
WORKDIR /server
EXPOSE 8080
COPY --from=build /bin/honeybadger-mcp-server .
ENTRYPOINT ["/server/honeybadger-mcp-server"]
CMD ["stdio"]
