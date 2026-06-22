FROM --platform=${BUILDPLATFORM} golang:1.26.4@sha256:792443b89f65105abba56b9bd5e97f680a80074ac62fc844a584212f8c8102c3 AS builder

WORKDIR /src

# Use the toolchain specified in go.mod, or newer
ENV GOTOOLCHAIN=auto

COPY go.mod go.sum .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
  go mod download && go mod verify

COPY cmd cmd
COPY internal internal

ARG TARGETARCH
ARG TARGETOS
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
  GOARCH=${TARGETARCH} GOOS=${TARGETOS} CGO_ENABLED=0 go build -a -ldflags="-s -w" -o wg-tunnel cmd/tunnel/*.go

FROM scratch AS export

COPY --from=builder /src/wg-tunnel wg-tunnel

FROM export

ENTRYPOINT ["/wg-tunnel"]
