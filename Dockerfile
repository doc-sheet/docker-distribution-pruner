FROM docker.io/golang:1.21-alpine as build

WORKDIR /build
COPY cmds cmds
COPY experimental experimental
COPY _support _support
COPY go.mod go.sum ./

ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOOS=linux

#	&& go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2 \
#	&& golangci-lint run \
RUN set -eux \
	&& go build -ldflags "-s -w" -trimpath ./cmds/docker-distribution-pruner \
	&& go test ./... \
	&& _support/go-fmt-check.sh

FROM docker.io/golang:1.21-alpine as main
ENV EXPERIMENTAL=true
COPY --from=build /build/docker-distribution-pruner /usr/local/bin/docker-distribution-pruner

ENTRYPOINT ["/usr/local/bin/docker-distribution-pruner"]
