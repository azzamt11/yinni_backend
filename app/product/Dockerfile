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

COPY --from=builder /src/bin/product /app/server

# 🔴 THIS WAS MISSING
COPY ./configs /data/conf

# 🔴 Copy dataset
COPY app/product/product_dataset.json /app/product_dataset.json

RUN chmod +x /app/server

EXPOSE 8002
EXPOSE 9002

VOLUME /data/conf

CMD ["./server", "-conf", "/data/conf"]
