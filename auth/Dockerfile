# auth/Dockerfile
FROM golang:1.25 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ВАЖНО: Компилируем статически для Linux
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -a -ldflags '-extldflags "-static"' -o main ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Копируем файлы
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static

# Делаем исполняемым (на всякий случай)
RUN chmod +x main

EXPOSE 8080
CMD ["./main"]
