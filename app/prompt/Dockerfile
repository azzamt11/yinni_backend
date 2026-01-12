# ---------- Build stage ----------
FROM golang:1.25.5 AS builder

WORKDIR /src
COPY . .

RUN GOPROXY=https://goproxy.cn make build

# ---------- Runtime stage ----------
FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    netbase \
    gettext-base \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /src/bin/prompt /app/server

COPY ./configs /data/conf

RUN chmod +x /app/server

EXPOSE 8003
EXPOSE 9003

VOLUME /data/conf

CMD ["./server", "-conf", "/data/conf"]
