# Build the manager binary

FROM docker.io/golang:1.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0-devel

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN sed -i "s|PAAS_VERSION = .*|PAAS_VERSION = \"$VERSION\"|" internal/version/main.go && \
    cat internal/version/main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -o manager cmd/manager/main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -o crypttool ./cmd/crypttool && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -o webservice ./cmd/webservice

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

LABEL MAINTAINER=belastingdienst
WORKDIR /
COPY --from=builder /workspace/manager /workspace/crypttool /workspace/webservice ./

ENTRYPOINT ["/manager"]
