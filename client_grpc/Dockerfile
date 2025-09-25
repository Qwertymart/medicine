# Stage 1: Builder
FROM golang:1.25.1 AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /client
COPY client_grpc/go.mod client_grpc/go.sum ./
RUN go mod download

COPY client_grpc .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o ctg_client main.go

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /client
COPY --from=builder /client/ctg_client .

USER nobody
EXPOSE 0

CMD ["./ctg_client"]
