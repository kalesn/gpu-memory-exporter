package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	gpuUsage = prometheus.NewDesc(
		"gpu_memory_usage",
		"GPU memory usage per process",
		[]string{"pid", "service", "pod"},
		nil)
)

func main() {
	// 注册指标
	mc := &MetricsCollector{Name: "GPU-MEMORY-EXPORTER"}
	prometheus.MustRegister(mc)

	// 启动 HTTP 服务
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}

type MetricsCollector struct {
	Name string
}

func (mc *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	log.Printf("MetricsCollector.collect.called")
	processes, err := GetAllRunningProcesses()
	if err != nil {
		log.Println(err)
		panic(err)
	}
	var once sync.Once
	for _, process := range processes {
		if !IsInSlice(process.Pid) {
			once.Do(func() {
				if err := GetContainerInfo(); err != nil {
					log.Println(err)
				}
			})
		}
		pid := strconv.Itoa(process.Pid)

		hostname, err := GetContainerHostname(process.Pid)
		if err != nil {
			log.Println(err)
			continue
		}
		serviceName := getServiceName(hostname)
		ch <- prometheus.MustNewConstMetric(gpuUsage,
			prometheus.GaugeValue, float64(process.Used), pid, serviceName, hostname)
	}
}

func (mc *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	log.Printf("MyCollector.Describe.called")
	ch <- gpuUsage
}

type ProcessInfo struct {
	Pid  int
	Used int
}

var processesInfos []*ProcessInfo

// GetAllRunningProcesses 获取当前正在使用GPU的进程ID(PID)
func GetAllRunningProcesses() ([]*ProcessInfo, error) {
	// Initialize NVML library
	err := nvml.Init()
	defer func() {
		_ = nvml.Shutdown()
	}()

	if err != nil {
		log.Println("Failed to initialize NVML:", err)
		return nil, err
	}
	// clear processesInfos
	processesInfos = processesInfos[0:0]

	// Get the number of GPUs in the system
	count, err := nvml.GetDeviceCount()
	if err != nil {
		log.Println("Failed to get GPU count:", err)
		return nil, err
	}

	for i := uint(0); i < count; i++ {
		// Get GPU handle
		gpu, err := nvml.NewDeviceLite(i)
		if err != nil {
			log.Printf("Failed to get handle for GPU %d: %v\n", i, err)
			continue
		}

		// Get list of processes running on this GPU
		processes, err := gpu.GetAllRunningProcesses()
		if err != nil {
			log.Printf("Failed to get processes for GPU %d: %v\n", i, err)
			continue
		}

		log.Printf("GPU %d processes:\n", i)
		for _, process := range processes {
			ProcessesInfo := &ProcessInfo{
				Pid:  int(process.PID),
				Used: int(process.MemoryUsed),
			}
			processesInfos = append(processesInfos, ProcessesInfo)
			log.Printf("\tProcess name: %s, PID: %d, Used memory: %d MB\n",
				process.Name, process.PID, process.MemoryUsed)
		}
	}
	return processesInfos, nil
}

// GetContainerHostname 根据PID获取container的主机名(POD Name)
func GetContainerHostname(pid int) (string, error) {
	if !IsInSlice(pid) {
		return "", errors.New(fmt.Sprintf("pid  %d is not the main process id", pid))
	}
	for _, info := range containerInfos {
		if info.Pid == pid {
			return info.Hostname, nil
		}
	}
	return "", errors.New(fmt.Sprintf("pid  %d is not in containerInfos", pid))
}

var PidSlice []int
var containerInfos []*ContainerInfo

type ContainerInfo struct {
	ID       string
	Pid      int
	Hostname string
}

// GetContainerInfo 获取所有运行的Container信息，uuid,PID,Hostname并进行关联
func GetContainerInfo() error {
	//ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Println(err)
		return err
	}

	containerList, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Println(err)
		return err
	}

	// clear slice
	PidSlice = PidSlice[0:0]
	containerInfos = containerInfos[0:0]

	// append containerInfo
	for _, container := range containerList {
		containerJson, err := cli.ContainerInspect(ctx, container.ID)
		if err != nil {
			panic(err)
		}
		PidSlice = append(PidSlice, containerJson.State.Pid)
		containerInfos = append(containerInfos, &ContainerInfo{
			ID:       container.ID,
			Pid:      containerJson.State.Pid,
			Hostname: containerJson.Config.Hostname,
		})
	}
	return nil
}

// 根据Container名称计算Service名称，以-为分隔符，除去后两段
func getServiceName(hostname string) string {
	HostnameSplit := strings.Split(hostname, "-")
	return strings.Join(HostnameSplit[:len(HostnameSplit)-2], "-")
}

// IsInSlice 判断Pid是否在切片中
func IsInSlice(item int) bool {
	for _, eachItem := range PidSlice {
		if item == eachItem {
			return true
		}
	}
	return false
}
