# web/Dockerfile

###################################
# 1. Депенденси
###################################
FROM node:22-alpine AS deps
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm install --legacy-peer-deps --ignore-scripts

###################################
# 2. Сборка
###################################
FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY web/ .

# Копируем proto-файлы из CTG_monitor в relative папку
COPY CTG_monitor/proto ./client_grpc/proto

# Проверим что файлы скопировались
RUN find ./client_grpc -name "*.proto" -type f || echo "Proto files not found in ./client_grpc"

# Создаем абсолютный путь /client_grpc для Next.js
RUN mkdir -p /client_grpc && cp -r ./client_grpc/proto /client_grpc/

# Проверим что файлы скопировались в абсолютный путь
RUN find /client_grpc -name "*.proto" -type f || echo "Proto files not found in /client_grpc"

ENV NEXT_PUBLIC_SKIP_PROTO_CHECK=true
RUN npm run build

###################################
# 3. Продакшен
###################################
FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV production

# Копируем собранное приложение
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json

# Создаем абсолютный путь для proto-файлов в рантайме
COPY --from=builder /client_grpc /client_grpc

EXPOSE 3000
CMD ["npm", "run", "start"]
