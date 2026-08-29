# StatSpatial UI (Vite) — StatGate Phase 10/44
FROM node:20-alpine AS builder

WORKDIR /app

ARG VITE_SPATIAL_API_URL=http://localhost:8108
ENV VITE_SPATIAL_API_URL=$VITE_SPATIAL_API_URL

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
