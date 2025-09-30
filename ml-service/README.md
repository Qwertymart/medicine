
# Инструкция по использованию ML-Service

## Описание

ML-Service — это веб-сервис для предиктивного анализа данных кардиотокографии (CTG). Сервис вычисляет статистические фичи из данных мониторинга плода и выполняет машинное обучение для прогнозирования риска гипоксии на различные временные периоды.

## Быстрый старт

### 1. Запуск сервиса

Убедитесь, что все сервисы запущены:

```bash
docker-compose up -d
```

Проверьте статус ML-Service:

```bash
curl http://localhost:8052/api/v1/ml/health
```

### 2. Доступ к Swagger UI

Откройте в браузере: http://localhost:8052/swagger/index.html

Здесь вы найдете интерактивную документацию API с возможностью тестирования эндпоинтов.

## API Эндпоинты

### GET /api/v1/ml/health

**Назначение:** Проверка статуса сервиса

**Пример запроса:**
```bash
curl http://localhost:8052/api/v1/ml/health
```

**Пример ответа:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-30T20:00:00Z"
}
```

### POST /api/v1/ml/features

**Назначение:** Вычисление статистических фичей из CTG данных

**Параметры запроса:**
- `card_id` (string, обязательно) - UUID карты пациента
- `target_time` (integer, обязательно) - Время анализа в секундах от начала записи

**Пример запроса:**
```bash
curl -X POST http://localhost:8052/api/v1/ml/features \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "target_time": 960
  }'
```

**Пример ответа:**
```json
{
  "card_id": "550e8400-e29b-41d4-a716-446655440000",
  "t_sec": 960,
  "fs_hz": 8.0,
  "available_windows": ["240s", "600s", "900s"],
  "features": {
    "f_240s_fhr_mean": 122.89,
    "f_240s_fhr_std": 9.53,
    "f_240s_fhr_min": 102.59,
    "f_240s_fhr_max": 194.05,
    "f_240s_uc_mean": 10.07,
    "f_240s_uc_std": 8.41
    // ... остальные 50+ фичей
  }
}
```

### POST /api/v1/ml/predict

**Назначение:** Полный предиктивный анализ (вычисление фичей + ML прогнозирование)

**Параметры запроса:** те же, что и для `/features`

**Пример запроса:**
```bash
curl -X POST http://localhost:8052/api/v1/ml/predict \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "target_time": 960
  }'
```

**Пример ответа:**
```json
{"ok":true,"card_id":"550e8400-e29b-41d4-a716-446655440000","t_sec":960,"ran":["trend5_trend","h15","h30","h45","h60"],"missing":{},"notes":[],"result":{"h15":{"pred":1,"proba":0.011491671627682765,"thr":0.000036194938259189004},"h30":{"pred":1,"proba":0.032751383958390744,"thr":0.0004490541518546731},"h45":{"pred":1,"proba":0.0856492378484748,"thr":0.0002933046029271888},"h60":{"pred":1,"proba":0.13731611394178808,"thr":0.001052385237199157},"trend5_trend":{"class":"down","proba":{"down":0.5571035571067863,"flat":0.2186890748566962,"up":0.22420736803651756}}},"ui":{"h15":{"pred":1,"risk_pct":1.15,"thr":0.000036194938259189004},"h30":{"pred":1,"risk_pct":3.28,"thr":0.0004490541518546731},"h45":{"pred":1,"risk_pct":8.56,"thr":0.0002933046029271888},"h60":{"pred":1,"risk_pct":13.73,"thr":0.001052385237199157},"summary":{"clinical_decision":"Рекомендуется усиленное наблюдение. Умеренный риск осложнений.","forecasts":["Вероятность развития гипоксии плода в следующие 15 минут: 1.1%","Вероятность развития гипоксии плода в следующие 30 минут: 3.3%","Вероятность развития гипоксии плода в следующие 45 минут: 8.6%","Вероятность развития гипоксии плода в следующие 60 минут: 13.7%"],"ok_60m":86.27,"risk_60m":13.73,"risk_category":"повышенный","text":"ПРЕДИКТИВНЫЙ АНАЛИЗ СОСТОЯНИЯ ПЛОДА\n\nКраткосрочные прогнозы:\n• Вероятность развития гипоксии плода в следующие 15 минут: 1.1%\n• Вероятность развития гипоксии плода в следующие 30 минут: 3.3%\n• Вероятность развития гипоксии плода в следующие 45 минут: 8.6%\n• Вероятность развития гипоксии плода в следующие 60 минут: 13.7%\n\nОбщий уровень риска: повышенный\nКлиническое заключение: Рекомендуется усиленное наблюдение. Умеренный риск осложнений."},"trend_probs":{"повышение":22.4,"снижение":55.7,"стабильное":21.9},"trend_text":"Тенденция изменения показателей (5 мин): снижение, уверенность средняя (56%)"}}
```

## Интерпретация результатов

### Тренд (trend5_trend)
- `down` - снижение показателей
- `flat` - стабильное состояние
- `up` - повышение показателей

### Риски (h15, h30, h45, h60)
- `proba` - вероятность риска гипоксии (от 0 до 1)
- `pred` - бинарное предсказание (1 = риск, 0 = норма)
- `thr` - пороговое значение модели

### UI поля
- `trend_text` - описание тренда на русском языке
- `summary.text` - полное медицинское заключение
- `clinical_decision` - клиническая рекомендация
- `risk_category` - категория риска

## Типовые сценарии использования

### Сценарий 1: Мониторинг в реальном времени

```bash
# Получение предсказания для текущего момента
curl -X POST http://localhost:8052/api/v1/ml/predict \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "your-patient-uuid",
    "target_time": 1200
  }'
```

### Сценарий 2: Анализ исторических данных

```bash
# Анализ состояния на 10-й минуте записи
curl -X POST http://localhost:8052/api/v1/ml/predict \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "your-patient-uuid", 
    "target_time": 600
  }'
```

### Сценарий 3: Получение только фичей для исследования

```bash
# Только вычисление фичей без ML
curl -X POST http://localhost:8052/api/v1/ml/features \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "your-patient-uuid",
    "target_time": 960
  }'
```

## Коды ошибок

- `400 Bad Request` - Неверные параметры запроса
- `404 Not Found` - Пациент с указанным card_id не найден
- `500 Internal Server Error` - Внутренняя ошибка сервиса или недоступность ML модели

## Требования к данным

1. **Карта пациента** должна существовать в базе данных
2. **CTG данные** должны содержать как минимум 4 минуты записи для вычисления фичей
3. **Частота дискретизации** должна быть 8 Гц
4. **target_time** не должен превышать длительность записи

## Логирование и диагностика

Для просмотра логов сервиса:

```bash
docker-compose logs -f ctg_ml_service
```

Для диагностики проблем с ML моделью:

```bash
# Проверка Python ML сервиса
curl http://localhost:8000/health

# Проверка Go ML шлюза
curl http://localhost:8052/api/v1/ml/health
```





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