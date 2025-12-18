FROM golang:1.25 AS builder

COPY . /src
WORKDIR /src

RUN go install github.com/google/wire/cmd/wire@latest

RUN GOPROXY=https://goproxy.cn go build -o /src/bin/server ./cmd/yinni_backend/main.go

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./server", "-conf", "/data/conf"]
