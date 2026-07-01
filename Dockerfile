FROM golang:1.25.5-alpine AS build
WORKDIR /build
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o /bin/honeybadger-mcp-server ./cmd/honeybadger-mcp-server

FROM alpine:3
WORKDIR /server
EXPOSE 8080
COPY --from=build /bin/honeybadger-mcp-server .
ENTRYPOINT ["/server/honeybadger-mcp-server"]
CMD ["stdio"]
