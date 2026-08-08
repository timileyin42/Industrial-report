# Vite bakes VITE_* env vars into the JS bundle at build time, not at
# container start — so they arrive as build args here, not runtime env.
# Rebuild the image (not just restart the container) to change either.
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_BASE_URL
ARG VITE_GOOGLE_MAPS_API_KEY
ARG VITE_MQTT_PUBLIC_BROKER_URL
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_GOOGLE_MAPS_API_KEY=${VITE_GOOGLE_MAPS_API_KEY}
ENV VITE_MQTT_PUBLIC_BROKER_URL=${VITE_MQTT_PUBLIC_BROKER_URL}
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
