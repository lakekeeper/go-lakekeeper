

FROM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=latest-main
ARG GIT_COMMIT=""
ARG GIT_TREE_STATE=""
ARG BUILD_DATE=""

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

# print platform for debug
RUN echo "Building for $GOOS/$GOARCH"

RUN go build \
    -ldflags "-X github.com/lakekeeper/go-lakekeeper/pkg/version.version=${VERSION} \
              -X github.com/lakekeeper/go-lakekeeper/pkg/version.gitCommit=${GIT_COMMIT} \
              -X github.com/lakekeeper/go-lakekeeper/pkg/version.gitTreeState=${GIT_TREE_STATE} \
              -X github.com/lakekeeper/go-lakekeeper/pkg/version.buildDate=${BUILD_DATE}" \
    -o lkctl ./cmd

FROM docker.io/debian:trixie-slim

ENV HOME="/" 

RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ca-certificates jq

RUN apt-get clean \
    && rm -rf /var/lib/apt/lists /var/cache/apt/archives

COPY --chmod=555 --from=builder /app/lkctl /usr/local/bin/lkctl

USER 1001
ENTRYPOINT ["/usr/local/bin/lkctl"]
CMD [ "--help" ]
