FROM node:20-alpine AS frontend

WORKDIR /app/web
ARG NEXT_PUBLIC_APP_VERSION=dev
ARG NEXT_PUBLIC_GITHUB_REPO=https://github.com/wang5339/octopus-2
ENV NEXT_PUBLIC_APP_VERSION=${NEXT_PUBLIC_APP_VERSION}
ENV NEXT_PUBLIC_GITHUB_REPO=${NEXT_PUBLIC_GITHUB_REPO}
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build

FROM golang:1.24-alpine AS backend

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/web/out ./static/out

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
