# Инструкция по использованию ML сервисов

## Содержание
1. [Python ML сервис (temp-ml)](#python-ml-сервис-temp-ml)
2. [Go ML шлюз (ml-service)](#go-ml-шлюз-ml-service)
3. [Запуск через Docker Compose](#запуск-через-docker-compose)

---

## Python ML сервис (temp-ml)

### Endpoints
- **GET** `/health` — проверка состояния сервиса
- **GET** `/meta` — метаданные моделей
- **POST** `/infer?verbose=true` — инференс на основе json-файла с фичами

### Пример запроса cURL
```bash
# Проверка состояния
curl http://localhost:8000/health

# Получение метаданных
curl http://localhost:8000/meta | jq

# Инференс
curl -X POST "http://localhost:8000/infer?verbose=true" \
  -H "Content-Type: application/json" \
  -d @sample_infer.json | jq
```

### Формат sample_infer.json
```json
{
  "card_id": "550e8400-e29b-41d4-a716-446655440000",
  "t_sec": 960,
  "fs_hz": 8,
  "available_windows": ["240s","600s","900s"],
  "features": {
    "f_240s_fhr_mean": 122.9,
    "f_600s_fhr_mean": 125.6,
    "f_900s_fhr_mean": 125.3,
    ... // остальные 50+ фич
  }
}
```

### Пример ответа
```json
{
  "ok": true,
  "card_id": "550e8400-e29b-41d4-a716-446655440000",
  "t_sec": 960,
  "ran": ["trend5_trend","h15","h30","h45","h60"],
  "result": { ... },
  "ui": {
    "trend_text": "Тенденция изменения показателей (5 мин): снижение, уверенность средняя (56%)",
    "summary": {
      "text": "ПРЕДИКТИВНЫЙ АНАЛИЗ СОСТОЯНИЯ ПЛОДА...",
      "clinical_decision": "Показатели в норме...",
      "risk_category": "низкий",
      "forecasts": [ ... ]
    }
  }
}
```

---

## Go ML шлюз (ml-service)

### Endpoints
- **GET** `/api/v1/ml/health` — проверка статуса шлюза
- **POST** `/api/v1/ml/features` — вычисление фичей без ML
- **POST** `/api/v1/ml/predict` — полный pipeline (фичи + инференс)

### Пример cURL запросов
```bash
# Проверка состояния шлюза
curl http://localhost:8052/api/v1/ml/health | jq

# Только фичи
curl -X POST http://localhost:8052/api/v1/ml/features \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "target_time": 960
  }' | jq

# Полное предсказание
curl -X POST http://localhost:8052/api/v1/ml/predict \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "target_time": 960
  }' | jq
```

### Формат запросов
```json
# /features
{
  "card_id": "...",
  "target_time": 960
}

# /predict
{
  "card_id": "...",
  "target_time": 960
}
```

### Пример ответа `/predict`
```json
{
  "ok": true,
  "card_id": "...",
  "t_sec": 960,
  "ran": [...],
  "result": {...},
  "ui": { ... }
}
```

---

## Запуск через Docker Compose
```bash
# Поднять все сервисы
docker-compose up -d

# Просмотр логов
docker-compose logs -f temp-ml ctg_ml_service
```

Файл сохранён как `README.md`.