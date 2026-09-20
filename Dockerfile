# syntax=docker/dockerfile:1

FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/grocery-router ./cmd/grocery-router

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 65532 grocery-router \
    && adduser -D -H -u 65532 -G grocery-router grocery-router \
    && mkdir -p /app/web/dist /data \
    && chown -R grocery-router:grocery-router /app /data
WORKDIR /app
COPY --from=api /out/grocery-router /usr/local/bin/grocery-router
COPY --from=web /src/web/dist ./web/dist
COPY --chown=grocery-router:grocery-router corpus ./corpus
COPY --chown=grocery-router:grocery-router archive ./archive
COPY --chown=grocery-router:grocery-router sources ./sources
COPY --chown=grocery-router:grocery-router scripts/container-entrypoint.sh /usr/local/bin/container-entrypoint
USER grocery-router:grocery-router
ENV GROCERY_ROUTER_ADDRESS=0.0.0.0:8080 \
    GROCERY_ROUTER_DATABASE=/data/grocery-router.db \
    GROCERY_ROUTER_ROOT=/app \
    GROCERY_ROUTER_WEB_ROOT=/app/web/dist
EXPOSE 8080
ENTRYPOINT ["container-entrypoint"]
