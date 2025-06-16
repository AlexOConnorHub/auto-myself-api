
FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/server /server
CMD ["/server"]
