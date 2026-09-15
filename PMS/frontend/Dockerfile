# Build stage
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
ARG VITE_PMS_API_URL=http://localhost:8091
ENV VITE_PMS_API_URL=$VITE_PMS_API_URL
ARG VITE_SPATIAL_API_URL=http://localhost:8108
ENV VITE_SPATIAL_API_URL=$VITE_SPATIAL_API_URL
RUN npm run build

# Serve with nginx
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
