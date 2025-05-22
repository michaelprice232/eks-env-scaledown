# Cross compilation multi-architecture build
# https://docs.docker.com/build/building/multi-platform/#cross-compiling-a-go-application
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

# These are made available when using the --platform docker build parameter, along with BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o /usr/local/bin/app ./cmd/main.go

FROM scratch

COPY --from=builder /usr/local/bin/app /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/app"]