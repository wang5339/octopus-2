FROM node:20-alpine AS frontend

WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build

FROM golang:1.24-alpine AS backend

RUN apk add --no-cache git python3

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/web/out ./static/out

# 更新价格数据
RUN python3 scripts/updatePrice.py || true

# 构建二进制
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -o octopus \
    -ldflags="-X 'github.com/bestruirui/octopus/internal/conf.Version=${VERSION}' \
              -X 'github.com/bestruirui/octopus/internal/conf.Commit=${COMMIT}' \
              -s -w" \
    -tags=jsoniter ./

FROM alpine:latest

ENV TZ=Asia/Shanghai

RUN apk add --no-cache ca-certificates tzdata su-exec && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    rm -rf /var/cache/apk/*

WORKDIR /app

COPY --from=backend /app/octopus ./octopus
COPY scripts/dockerfiles/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh ./octopus

EXPOSE 8080

VOLUME ["/app/data"]

CMD ["/entrypoint.sh"]
