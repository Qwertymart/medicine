# Сервис предиктивной аналитики физиологических данных

## Архитектура системы (дерево)

```
service-root
├── web
│   └── React
├── ctg-monitor
│   └── Сердце проекта - Основной backend
├── medicine_card
│   └── HTTP/gRPC-сервис хранения данных
├── medicine_emulator
│   └── Генерация тестовых данных (гипоксия/норма)
├── ml-service
│   └── Предиктивная аналитика (модель ML)
├── mqtt-broker
│   └── Mosquitto
├── database
    └── PostgreSQL

```

## Инструкция по развертыванию

### Требования

- Docker и Docker Compose
- Git


### Шаг 1. Клонирование репозитория

```bash
git clone https://github.com/Qwertymart/medicine.git
```


```

### Шаг 2. Запуск через Docker Compose

```bash
docker-compose up -d
```

