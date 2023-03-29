FROM dockerimg.lenztechretail.com/infra/golang:1.18-gpu as builder

WORKDIR /app

RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o gpu-memory-exporter main.go

FROM dockerimg.lenztechretail.com/infra/dcgm-exporter:2.4.6-2.6.9-ubuntu20.04

WORKDIR /app

COPY --from=builder /app/gpu-memory-exporter .

EXPOSE 8080
ENTRYPOINT ["./gpu-memory-exporter"]
