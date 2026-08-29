FROM node:20-alpine AS build
WORKDIR /app
ARG VITE_TURN_URL
ARG VITE_TURN_USERNAME
ARG VITE_TURN_CREDENTIAL
ENV VITE_TURN_URL=${VITE_TURN_URL}
ENV VITE_TURN_USERNAME=${VITE_TURN_USERNAME}
ENV VITE_TURN_CREDENTIAL=${VITE_TURN_CREDENTIAL}
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
RUN apk add --no-cache wget
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
