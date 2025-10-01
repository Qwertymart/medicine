# Dockerfile для Python ML сервиса (temp_ml)

FROM python:3.11-slim

# Установка системных зависимостей
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Установка рабочей директории
WORKDIR /app

# Копирование файлов requirements
COPY requirements.txt .

# Установка Python зависимостей
RUN pip install --upgrade pip
RUN pip install --upgrade --no-cache-dir -r requirements.txt

# Копирование исходного кода
COPY . .

# Создание директории для моделей
RUN mkdir -p /app/out

# Установка переменных окружения
ENV PYTHONPATH=/app
ENV OUT_DIR=/app/out
ENV PORT=8000

# Открытие порта
EXPOSE 8000

# Команда запуска
CMD ["python", "app.py"]