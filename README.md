# Сервис предиктивной аналитики физиологических данных

## Архитектура системы (дерево)

```
service-root
├── web
│   └── React (обслуживается Nginx, порт 3000)
├── api-gateway
│   └── Envoy/Nginx (порт 80/443)
├── ctg-monitor
│   └── Go gRPC/HTTP (порт 50051 gRPC, порт 8080 HTTP)
├── medicine_card
│   └── Go HTTP/gRPC (порт 50052 gRPC, порт 8081 HTTP)
├── medicine_emulator
│   └── Python (порт 8082 HTTP, генерирует MQTT-сообщения)
├── temp-ml
│   └── Python FastAPI (порт 8000 HTTP)
├── ml-service
│   └── Go HTTP (порт 8052 HTTP)
├── mqtt-broker
│   └── Mosquitto (порт 1883 MQTT, порт 9001 WebSocket)
├── database
│   └── PostgreSQL (порт 5432)
└── monitoring
    ├── Prometheus (порт 9090)
    └── Grafana (порт 3001)
```

## Технологический стек

- **Языки и фреймворки**: Go (gRPC, net/http), Python (FastAPI), React
- **Коммуникации**: gRPC, HTTP/REST, MQTT
- **Базы данных**: PostgreSQL
- **Контейнеризация**: Docker, Docker Compose
- **Оркестрация (опционально)**: Kubernetes
- **Мониторинг**: Prometheus, Grafana
- **API документация**: Swagger/OpenAPI для HTTP-сервисов
- **Тестирование**:
  - Unit-тесты (Go, pytest)
  - Integration-тесты с Testcontainers
  - Linter (golangci-lint, flake8)

## Порты и эндпоинты

| Сервис            | Протокол | Порт  | Swagger UI                 |
|-------------------|----------|-------|----------------------------|
| web               | HTTP     | 3000  | n/a                        |
| api-gateway       | HTTP     | 80,443| n/a                        |
| ctg-monitor       | gRPC     | 50051 | n/a                        |
| ctg-monitor       | HTTP     | 8080  | `/swagger/index.html`      |
| medicine_card     | gRPC     | 50052 | n/a                        |
| medicine_card     | HTTP     | 8081  | `/swagger/index.html`      |
| medicine_emulator | HTTP     | 8082  | `/docs` (FastAPI)          |
| temp-ml           | HTTP     | 8000  | `/docs` (FastAPI)          |
| ml-service        | HTTP     | 8052  | `/swagger/index.html`      |
| mqtt-broker       | MQTT     | 1883  | n/a                        |
| mqtt-broker       | WS       | 9001  | n/a                        |
| postgres          | TCP      | 5432  | n/a                        |
| prometheus        | HTTP     | 9090  | `/graph`                   |
| grafana           | HTTP     | 3001  | `/`                        |

## Инструкция по развертыванию

### Требования

- Docker >=20.10, Docker Compose >=1.29
- Git >=2.25
- Node.js >=16 (для сборки web)
- Python 3.9+ (для temp-ml и emulator)
- Go 1.18+ (для сервисов на Go)
- Доступ к PostgreSQL (или локальный контейнер)

### Шаг 1. Клонирование репозитория

```bash
git clone https://github.com/Qwertymart/medicine.git && cd medicine
```

### Шаг 2. Настройка переменных окружения

Создать файл `.env`:

```env
DB_HOST=postgres
DB_PORT=5432
DB_USER=<user>
DB_PASSWORD=<password>
DB_NAME=medicine
MQTT_HOST=mosquitto
MQTT_PORT=1883
MQTT_USER=<user>
MQTT_PASSWORD=<password>
LOG_LEVEL=info
```

### Шаг 3. Сборка и запуск сервисов

```bash
docker-compose up --build -d
```

### Шаг 4. Проверка готовности и тестирование

1. **Healthchecks**: все сервисы имеют endpoint `/health` или healthcheck в Compose.
2. **Swagger/UI**:
   - HTTP-сервисы: перейти по `http://localhost:{порт}/swagger/index.html` или `/docs`.
