# syntax=docker/dockerfile:1

FROM golang:1.25 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /server

FROM alpine:latest AS certificates
RUN apk --no-cache add ca-certificates

FROM scratch

COPY --from=builder /server /server
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 65532:65532

ENTRYPOINT ["/server"]
