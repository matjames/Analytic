FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
# Install node dependencies with legacy peer deps; npm install updates package-lock when package.json and lockfile drift
RUN npm install --legacy-peer-deps
COPY . .
ARG REACT_APP_API_BASE_URL_DEV=http://localhost:9090/api
ARG REACT_APP_API_BASE_URL_PROD=http://localhost:9090/api
ENV REACT_APP_API_BASE_URL_DEV=$REACT_APP_API_BASE_URL_DEV
ENV REACT_APP_API_BASE_URL_PROD=$REACT_APP_API_BASE_URL_PROD
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/build /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
