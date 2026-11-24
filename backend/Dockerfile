# Stage 1: Build Vue 3 SPA (Pendekatan A)
FROM node:22-alpine AS web-builder
WORKDIR /web
COPY web-gacha/package*.json ./
RUN npm ci
COPY web-gacha/ ./
RUN npm run build

# Stage 2: Build static Go binary
FROM golang:1.27-alpine AS go-builder
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gacha-server ./cmd/server

# Stage 3: Non-root OCP-compatible runtime
FROM alpine:3.21
RUN adduser -D -u 10001 -G root appuser
WORKDIR /app
COPY --from=go-builder /out/gacha-server /app/gacha-server
COPY --from=web-builder /web/dist /app/web-dist
COPY docs/api/openapi.yaml /app/docs/api/openapi.yaml
RUN chown -R 10001:0 /app && chmod -R g=u /app
USER 10001:0
EXPOSE 8080
ENTRYPOINT ["/app/gacha-server"]
