FROM golang:1.26.5-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
	-trimpath \
	-ldflags="-s -w" \
	-o /out/centralsystem \
	./cmd/centralsystem

FROM scratch

WORKDIR /app

COPY --from=build /out/centralsystem /centralsystem

VOLUME ["/app/data"]
EXPOSE 8080 8081

ENTRYPOINT ["/centralsystem"]
