FROM nvcr.io/nvidia/cuda:12.0.0-base-ubuntu20.04 as  builder
LABEL io.k8s.display-name="GPU MEMORY Exporter"

#
RUN sed -i -E "s/[a-zA-Z0-9]+.ubuntu.com/mirrors.aliyun.com/g" /etc/apt/sources.list
RUN apt-get update && apt-get install -y --no-install-recommends \
    datacenter-gpu-manager=1:2.4.6 libcap2-bin \
    gcc g++  make wget git curl  ca-certificates apt-transport-https && update-ca-certificates


# Required for DCP metrics
ENV NVIDIA_DRIVER_CAPABILITIES=compute,utility,compat32
# disable all constraints on the configurations required by NVIDIA container toolkit
ENV NVIDIA_DISABLE_REQUIRE="true"
ENV NVIDIA_VISIBLE_DEVICES=all

ENV NO_SETCAP=""

WORKDIR /app

# install GoPkg
RUN  wget --no-check-certificate  https://studygolang.com/dl/golang/go1.18.8.linux-amd64.tar.gz && tar xfz go1.18.8.linux-amd64.tar.gz

ENV  PATH ${PATH}:/app/go/bin

COPY go.mod .
COPY go.sum .

RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN go mod download

COPY . .

RUN go build -o gpu-memory-exporter main.go

FROM nvcr.io/nvidia/cuda:12.0.0-base-ubuntu20.04

WORKDIR /app

COPY --from=builder /app/gpu-memory-exporter .

EXPOSE 8080
ENTRYPOINT ["./gpu-memory-exporter"]