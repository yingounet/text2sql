# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git (required for go mod download)
RUN apk add --no-cache git

# 设置中国镜像源
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GOPRIVATE=

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Run stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /server .
COPY config.yaml .

# 数据目录
RUN mkdir -p /app/data

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 8080

CMD ["./server"]
