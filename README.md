# GPU Memory Exporter

该项目是一个用Go编写的Prometheus Exporter，用于抓取每个进程的GPU显存使用率并根据进程的pid找到对应的Docker主机名。它使用nvidia-smi命令获取GPU内存信息，然后将其作为指标公开给Prometheus。本项目还包含了一个Dockerfile，可以使用它构建一个包含该Exporters的Docker镜像。


# 运行方式

### 直接运行

`go run main.go`

### 使用Docker


- 构建Docker镜像

`docker build -t gpu-memory-exporter .`


- 运行Docker容器

`docker run -p 8080:8080 --pid=host --runtime=nvidia -v /var/run/docker.sock:/var/run/docker.sock gpu-memory-exporter`



# 暴露的指标

### 该Exporter暴露了以下指标：

- gpu_memory_usage: 每个进程的GPU内存使用率，包括进程的pid和Docker主机名。

### 注意
- 如果运行在k8s中，并且使用prometheus operator的service monitor 进行采集需要进行drop label操作，否则pod、service标签的值会被覆盖为gpu-memory-exporter
```
relabelings:
    - action: labeldrop
      regex: (pod|service)
```

# 要求

安装NVIDIA CUDA驱动程序和nvidia-docker2扩展


# 参考文献

- Promethues Client Golang：https://github.com/prometheus/client_golang
- NVIDIA Docker：https://github.com/NVIDIA/nvidia-docker
- Docker: https://github.com/docker/docker

