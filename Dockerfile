FROM golang:1.16 AS builder

WORKDIR /app

RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o gpu-memory-exporter main.go

FROM nvidia/cuda:10.0-base

WORKDIR /app

COPY --from=builder /app/gpu-memory-exporter .

EXPOSE 8080
CMD ["./gpu-memory-exporter"]
