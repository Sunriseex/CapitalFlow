FROM node:24-alpine AS build

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web ./
RUN npm run build

FROM nginx:1.29-alpine
ARG APP_VERSION=dev

LABEL org.opencontainers.image.title="CapitalFlow Web" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.source="https://github.com/Sunriseex/CapitalFlow"

COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/web/dist /usr/share/nginx/html

EXPOSE 8080
