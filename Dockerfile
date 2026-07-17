# syntax=docker/dockerfile:1.18

FROM golang:1.26.5-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS=linux
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
	-trimpath \
	-buildvcs=false \
	-ldflags="-s -w -buildid=" \
	-o /out/centralsystem \
	./cmd/centralsystem

RUN mkdir -p /out/data && chown 65532:65532 /out/data

FROM scratch

WORKDIR /app

ARG CREATED
ARG VERSION=dev
ARG REVISION
ARG SOURCE="https://github.com/johnmaddison/ocpp-tester"
ARG HOMEPAGE="https://github.com/johnmaddison/ocpp-tester"

LABEL org.opencontainers.image.title="OCPP Tester" \
	org.opencontainers.image.description="Go-based central system for testing OCPP charge point behavior" \
	org.opencontainers.image.source="${SOURCE}" \
	org.opencontainers.image.url="${HOMEPAGE}" \
	org.opencontainers.image.documentation="${HOMEPAGE}#readme" \
	org.opencontainers.image.licenses="MIT" \
	org.opencontainers.image.revision="${REVISION}" \
	org.opencontainers.image.created="${CREATED}" \
	org.opencontainers.image.version="${VERSION}"

COPY --from=build /out/centralsystem /centralsystem
COPY --from=build --chown=65532:65532 /out/data /app/data

VOLUME ["/app/data"]
EXPOSE 8080 8081

USER 65532:65532
ENTRYPOINT ["/centralsystem"]
