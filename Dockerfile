# Build the manager binary

FROM docker.io/golang:1.21 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GOINSECURE="proxy.golang.org/*,github.com,github.com/*"
ARG GONOSUMDB="proxy.golang.org/*,github.com,github.com/*"
ARG GOPRIVATE="proxy.golang.org/*,github.com,github.com/*"
ARG VERSION=v0.0.0-devel

ARG cert_location=/usr/local/share/ca-certificates

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY cli/ cli/
COPY controllers/ controllers/
COPY internal/ internal/
COPY config/ config/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN sed -i "s|PAAS_VERSION = .*|PAAS_VERSION = \"$VERSION\"|" internal/version/main.go && \
    cat internal/version/main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -o manager main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -o crypttool ./cli/crypttool && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -o webservice ./cli/webservice

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

LABEL MAINTAINER=CPET
WORKDIR /
COPY --from=builder /workspace/manager /workspace/crypttool /workspace/webservice .

ENTRYPOINT ["/manager"]
