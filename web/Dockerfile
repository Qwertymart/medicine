# web/Dockerfile

############################################################
# ЭТАП 1: Установка зависимостей (кэшируется)
############################################################
FROM node:22-alpine AS deps
WORKDIR /app

# Копируем package.json и package-lock.json (если он есть)
# * не даст ошибку, если lock-файл отсутствует
COPY web/package*.json ./

# Используем `npm install`, так как `npm ci` требует наличия lock-файла
RUN npm install --legacy-peer-deps --ignore-scripts


############################################################
# ЭТАП 2: Сборка приложения
############################################################
FROM node:22-alpine AS builder
WORKDIR /app

# Копируем установленные зависимости
COPY --from=deps /app/node_modules ./node_modules

# Копируем исходный код фронтенда
COPY web/ .

# !! КЛЮЧЕВОЕ ИСПРАВЛЕНИЕ !!
# Создаем АБСОЛЮТНЫЙ путь /client_grpc/proto и копируем proto-файлы туда,
# куда смотрит код приложения во время сборки.
RUN mkdir -p /client_grpc/proto
COPY CTG_monitor/proto /client_grpc/proto

# Запускаем сборку Next.js
RUN npm run build


############################################################
# ЭТАП 3: Финальный Production-образ
############################################################
FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV production

# Копируем только необходимые для запуска файлы из сборщика
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/node_modules ./node_modules

# !! ВАЖНО !!
# Копируем proto-файлы в тот же АБСОЛЮТНЫЙ путь в финальном образе,
# так как они могут понадобиться и в рантайме.
COPY --from=builder /client_grpc /client_grpc

EXPOSE 3000
CMD ["npm", "run", "start"]
