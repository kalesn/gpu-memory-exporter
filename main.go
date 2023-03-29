package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//const (
//gpuMemRegexPattern = `(\d+)MiB \/ (\d+)MiB`
//pidRegexPattern = `(\d+)MiB \|$`
//)

var (
	gpuUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gpu_memory_usage",
			Help: "GPU memory usage per process",
		},
		[]string{"pid", "docker_hostname"},
	)
)

var PidSlice []int
var containerInfos []ContainerInfo

func init() {
	prometheus.MustRegister(gpuUsage)
}

type ContainerInfo struct {
	ID       string
	Pid      int
	Hostname string
}

func InSlice(item int) bool {
	for _, eachItem := range PidSlice {
		if item == eachItem {
			return true
		}
	}
	return false
}
func GetContainerHostname(pid int) (string, error) {
	if !InSlice(pid) {
		err := GetContainerInfo()
		if err != nil {
			return "", err
		}
	}
	for _, info := range containerInfos {
		if info.Pid == pid {
			return info.Hostname, nil
		}
	}
	return "", errors.New(fmt.Sprintf("pid  %d is not the main process id", pid))
}

func GetContainerInfo() error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.41"))
	if err != nil {
		panic(err)
	}

	containerList, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
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
		containerInfo := ContainerInfo{
			ID:       container.ID,
			Pid:      containerJson.State.Pid,
			Hostname: containerJson.Config.Hostname,
		}
		containerInfos = append(containerInfos, containerInfo)
	}
	return nil
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		for {
			//cmd := exec.Command("nvidia-smi")
			//out, err := cmd.Output()
			//if err != nil {
			//	panic(err)
			//}
			//
			//processes := strings.Split(string(out), "\n")
			//
			//for _, process := range processes[1:] {
			//	match := regexp.MustCompile(pidRegexPattern).FindStringSubmatch(process)
			//	if match != nil {
			//		processReal := strings.Fields(process)
			//		pid, _ := strconv.Atoi(processReal[4])
			//
			//		hostname, err := GetContainerHostname(pid)
			//		if err != nil {
			//			fmt.Println(err)
			//		}
			//		used, _ := strconv.Atoi(strings.TrimRight(processReal[7], "MiB"))
			//		gpuUsage.WithLabelValues(pid, hostname).Set(float64(used))
			//	}
			//}
			err := GetAllRunningProcesses()
			if err != nil {
				panic(err)
			}
			for _, process := range processesInfos {
				pid := process.Pid
				hostname, err := GetContainerHostname(pid)
				if err != nil {
					fmt.Println(err)
				}
				used := process.Used
				gpuUsage.WithLabelValues(strconv.Itoa(pid), hostname).Set(float64(used))
			}
			time.Sleep(time.Second * 1)
		}
	}()
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
