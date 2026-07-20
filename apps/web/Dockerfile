FROM node:24-alpine AS builder

WORKDIR /app

COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci

COPY apps/web/ ./
RUN npm run build

FROM nginx:1.27-alpine

# API_UPSTREAM: nama host backend API. Default "api" cocok buat docker-compose
# lokal (nama service compose). Di K8s, di-override lewat --build-arg karena
# nama Service-nya beda per environment (namespace/naming convention).
ARG API_UPSTREAM=api

COPY --from=builder /app/dist /usr/share/nginx/html
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
RUN sed -i "s|http://api:8080|http://${API_UPSTREAM}:8080|g" /etc/nginx/conf.d/default.conf

EXPOSE 80
