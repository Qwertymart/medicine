# Сервис предиктивной аналитики физиологических данных

## Архитектура системы

Система представляет собой набор взаимосвязанных сервисов и компонентов:

```
+-------------------+       +-------------+       +----------------+
| Пользовательский  | <---> | API Gateway | <---> | CTG Monitor    |
| интерфейс (web)   |       | (envoy/nginx)|      | (gRPC-сервис)  |
+-------------------+       +-------------+       +----------------+
        |                                         |
        v                                         v
+-------------------+       +-------------+    +----------------+
| Medicine Card     |       | ML-Service  |    | MQTT Broker    |
| сервис            | <---> | (предиктивная|    | (mosquitto)    |
+-------------------+       | аналитика)  |    +----------------+
        ^                    +-------------+            |
        |                         |                     |
+-------------------+            v                     v
| Medicine Emulator |       +-------------+      +----------------+
| (генерация данных)|       | Database    |      | Monitoring     |
+-------------------+       | (PostgreSQL)|      | (Prometheus)   |
                            +-------------+      +----------------+
```

### Компоненты

- **web**: фронтенд-приложение на React, обслуживается Nginx.
- **envoy/nginx**: шлюз API, маршрутизация и балансировка нагрузки.
- **CTG Monitor**: gRPC-сервис для сбора и обработки сигналов кардиотокографии.
- **medicine_card**: сервис хранения и предоставления истории физиологических данных.
- **medicine_emulator**: утилита-генератор тестовых данных (гипоксия и нормальный режим).
- **ml-service**: микросервис машинного обучения, выполняет предсказания на основе сигналов.
- **mosquitto**: MQTT-брокер для передачи сообщений между сервисами.
- **monitoring**: сбор метрик, интеграция с Prometheus и Grafana.
- **database**: PostgreSQL для хранения структурированных данных.

## Инструкция по развертыванию

### Требования

- Docker и Docker Compose
- Git
- Доступ к базе PostgreSQL
- (Опционально) Kubernetes-кластер и kubectl

### Шаг 1. Клонирование репозитория

```bash
git clone <REPO_URL> && cd <REPO_DIR>
```

### Шаг 2. Настройка переменных окружения

Создать файл `.env` в корне проекта со следующими параметрами:

```env
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=user
POSTGRES_PASSWORD=pass
POSTGRES_DB=medicine
MQTT_HOST=mosquitto
MQTT_PORT=1883
```

### Шаг 3. Запуск через Docker Compose

```bash
docker-compose up -d
```

- Сервис frontend доступен на http://localhost:3000
- gRPC-сервисы на портах 50051 (CTG Monitor) и 50052 (Medicine Card)
- ML-Service на порту 50053
- MQTT Broker на порту 1883

### Шаг 4. Инициализация базы данных

```bash
docker exec -it <postgres_container> psql -U user -d medicine -f init.sql
```

### (Опционально) Шаг 5. Развертывание в Kubernetes

1. Создать namespace:

   ```bash
   kubectl create namespace medicine
   ```

2. Установить секреты и ConfigMap:

   ```bash
   kubectl apply -f k8s/configmap.yaml -n medicine
   kubectl apply -f k8s/secret.yaml -n medicine
   ```

3. Развернуть манифесты:

   ```bash
   kubectl apply -f k8s/deployment/ -n medicine
   kubectl apply -f k8s/service/ -n medicine
   ```

4. Проверить статусы:

   ```bash
   kubectl get pods -n medicine
   kubectl get svc -n medicine
   ```
