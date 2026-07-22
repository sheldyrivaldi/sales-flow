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
# API_UPSTREAM_PORT: port backend API. Default 8080 (container port, cocok buat
# docker-compose lokal & K8s Service). Perlu di-override kalau API_UPSTREAM
# nunjuk ke host port yang beda dari container port-nya sendiri — misal deploy
# docker flat di satu host yang portnya udah dipakai service lain, sehingga API
# di-publish di host port lain (lihat deploymentType:docker di cicd.yml).
ARG API_UPSTREAM=api
ARG API_UPSTREAM_PORT=8080

COPY --from=builder /app/dist /usr/share/nginx/html
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
RUN sed -i "s|http://api:8080|http://${API_UPSTREAM}:${API_UPSTREAM_PORT}|g" /etc/nginx/conf.d/default.conf

EXPOSE 80
