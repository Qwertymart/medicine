# Dockerfile для CTG эмулятора
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Компилируем приложение из cmd/main.go
RUN go build -o ctg-emulator ./cmd/main.go

# Production образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Копируем скомпилированное приложение
COPY --from=builder /app/ctg-emulator .

# Копируем index.html из cmd/ в корень рабочей директории
COPY --from=builder /app/cmd/index.html .

# Копируем данные в правильную структуру для работы приложения
COPY --from=builder /app/data ./data

# Открываем порт на всех интерфейсах
EXPOSE 8081

# Запускаем приложение
CMD ["./ctg-emulator"]
