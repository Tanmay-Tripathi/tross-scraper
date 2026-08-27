# syntax=docker/dockerfile:1

# ---- Build ----
FROM node:22-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .
# Vite inlines env vars at build time, so the API host must be known here.
ARG VITE_APP_NAME="Tross Scraper"
ARG VITE_API_HOST=""
ARG VITE_LOG_ERRORS="false"
ENV VITE_APP_NAME=${VITE_APP_NAME} \
    VITE_API_HOST=${VITE_API_HOST} \
    VITE_LOG_ERRORS=${VITE_LOG_ERRORS}
RUN npm run build

# ---- Runtime ----
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
