FROM golang:1.24.4-alpine AS build
WORKDIR /build
COPY . .
RUN go build -o /bin/honeybadger-mcp-server ./cmd/honeybadger-mcp-server

FROM alpine:3
WORKDIR /server
COPY --from=build /bin/honeybadger-mcp-server .
ENTRYPOINT ["/server/honeybadger-mcp-server", "stdio"]
