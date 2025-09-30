Endpoints и примеры запросов вынесены в файл `README.md`.

```markdown
# Itelma ML Pipeline — API справочник
## Python ML сервис (`temp-ml`)
| Метод | URL | Назначение |
|-------|-----|------------|
| `GET` | `/health` | Проверка статуса сервиса |
| `GET` | `/meta` | Метаданные доступных моделей |
| `POST` | `/infer?verbose=true` | Инференс по одной точке фичей |

### cURL
```
# Быстрый пингcurl http://localhost:8000/health | jq

# Метаданныеcurl http://localhost:8000/meta | jq

# Инференс (файл sample_infer.json должен лежать рядом)curl -X POST "http://localhost:8000/infer?verbose=true" \
  -H "Content-Type: application/json" \
  -d @sample_infer.json
```

---

## Go ML-gateway (`ml-service`)
| Метод | URL | Назначение |
|-------|-----|------------|
| `GET`  | `/api/v1/ml/health`   | Проверка статуса шлюза |
| `POST` | `/api/v1/ml/features` | Вычислить фичи (без ML) |
| `POST` | `/api/v1/ml/predict`  | Фичи + вызов Python ML |

### cURL
```
# Здоровьеcurl http://localhost:8052/api/v1/ml/health | jq

# Только фичиcurl -X POST http://localhost:8052/api/v1/ml/features \
  -H "Content-Type: application/json" \
  -d '{
        "card_id": "550e8400-e29b-41d4-a716-446655440000",
        "target_time": 960
      }' | jq

# Полное предсказаниеcurl -X POST http://localhost:8052/api/v1/ml/predict \
  -H "Content-Type: application/json" \
  -d '{
        "card_id": "550e8400-e29b-41d4-a716-446655440000",
        "target_time": 960
      }' | jq
```

---

### Формат `/infer` (коротко)
```
{
  "card_id": "550e8400-e29b-41d4-a716-446655440000",
  "t_sec": 960,                 // момент времени (секунда записи)
  "fs_hz": 8,                   // частота дискретизации
  "available_windows": ["240s","600s","900s"],
  "features": { ... }           // 50+ фич (ключи f_{sec}s_* )
}
```

Ответ с `verbose=true` содержит поле `ui` с готовым русскоязычным вердиктом.
```
{
  "ok": true,
  "card_id": "...",
  "t_sec": 960,
  "result": { ... },
  "ui": {
    "trend_text": "Тенденция изменения показателей (5 мин): снижение, уверенность средняя (56%)",
    "summary": {
      "text": "ПРЕДИКТИВНЫЙ АНАЛИЗ СОСТОЯНИЯ ПЛОДА ...",
      ...
    }
  }
}
```
Файл сохранён в репозитории.