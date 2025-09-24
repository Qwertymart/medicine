# Этап сборки
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install --production
COPY . .
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV production
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs && \
    mkdir -p /tmp/.npm
ENV NPM_CONFIG_CACHE=/tmp/.npm
ENV NPM_CONFIG_UPDATE_NOTIFIER=false
COPY --from=builder --chown=nextjs:nodejs /app ./
USER nextjs
# EXPOSE 3000
# ENV PORT 3000
CMD ["npm", "run", "start"]