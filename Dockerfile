ARG GO_VERSION=1.23
ARG ALPINE_VERSION=3.20

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

ARG GOARCH=amd64
ARG GOARM
ARG VERSION

WORKDIR /tmp/gobuild
COPY ./ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} \
    go build -a -mod=vendor \
    -ldflags="-s -w \
    -X main.commit=${VCS_REF} \
    -X main.version=${VERSION} \
    -X main.buildSource=Docker" \
    -o /tmp/gobuild/lazypodman .

# Podman ships its own CLI and a working compose provider, so we use the
# official Podman image as the runtime. Mount your Podman socket (and set
# CONTAINER_HOST to its in-container path) to manage containers from here.
FROM quay.io/podman/stable:latest

LABEL \
    org.opencontainers.image.authors="lazypodman" \
    org.opencontainers.image.url="https://github.com/lazypodman/lazypodman" \
    org.opencontainers.image.documentation="https://github.com/lazypodman/lazypodman" \
    org.opencontainers.image.source="https://github.com/lazypodman/lazypodman" \
    org.opencontainers.image.title="lazypodman" \
    org.opencontainers.image.description="The lazier way to manage everything podman"

ENV CONTAINER_HOST=unix:///run/podman/podman.sock

COPY --from=builder /tmp/gobuild/lazypodman /usr/local/bin/lazypodman

ENTRYPOINT [ "/usr/local/bin/lazypodman" ]
