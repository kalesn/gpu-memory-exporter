FROM golang:1.16-alpine AS builder

RUN apk add --no-cache git libc-dev gcc libgcc libstdc++ nvidia-cuda-dev=11.3.0-r0 nvidia-cuda-toolkit=11.3.0-r0

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o gpu-memory-exporter main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates libstdc++ nvidia-cuda-runtime=11.3.0-r0

WORKDIR /app

COPY --from=builder /app/gpu-memory-exporter .

EXPOSE 8080
CMD ["./gpu-memory-exporter"]
