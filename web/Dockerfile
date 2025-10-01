# web/Dockerfile.dev - Простая версия для разработки
FROM node:22-alpine

WORKDIR /app

# Копируем package files
COPY web/package*.json ./

# Устанавливаем ВСЕ зависимости (включая dev)
RUN npm install --legacy-peer-deps

# Копируем proto файлы
RUN mkdir -p /client_grpc/proto
COPY CTG_monitor/proto /client_grpc/proto

# Настройки для hot reload
ENV WATCHPACK_POLLING=true
ENV NODE_ENV=development

EXPOSE 3000

# Запускаем dev сервер
CMD ["npm", "run", "dev"]