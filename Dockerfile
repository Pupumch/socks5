FROM golang:1.25 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOARM=$TARGETVARIANT

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/socks-server ./cmd/proxy

FROM scratch

COPY --from=builder /bin/socks-server /socks-server

ENTRYPOINT ["/socks-server"]