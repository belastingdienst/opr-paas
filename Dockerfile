# Build the manager binary

FROM --platform=${BUILDPLATFORM} docker.io/golang:1.24 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=v0.0.0-devel

WORKDIR /workspace

# Copy the go source
COPY . .

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -v -a -ldflags="-X 'github.com/belastingdienst/opr-paas/v3/internal/version.PaasVersion=${VERSION}'" -o manager ./cmd/manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

LABEL MAINTAINER=belastingdienst
WORKDIR /
COPY --from=builder /workspace/manager ./

ENTRYPOINT ["/manager"]
