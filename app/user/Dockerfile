# ---------- Build stage ----------
FROM golang:1.24 AS builder

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

# 👇 COPY THE CORRECT OUTPUT
COPY --from=builder /src/bin/user /app/server

RUN chmod +x /app/server

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./server", "-conf", "/data/conf/config.runtime.yaml"]
