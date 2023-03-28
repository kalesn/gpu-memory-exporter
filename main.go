package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	//gpuMemRegexPattern = `(\d+)MiB \/ (\d+)MiB`
	//pidRegexPattern    = `(\d+)MiB \|$`
	pidRegexPattern = `(\d+)MiB \|$`
)

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
	return "", errors.New("pid is not the main process id")
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
		//fmt.Printf("Container ID: %s, Pid: %d\n", container.ID, container.Pid)
	}
	return nil
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		for {
			cmd := exec.Command("nvidia-smi")
			out, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
			}

			processes := strings.Split(string(out), "\n")

			for _, process := range processes[1:] {
				fmt.Println(process)
				match := regexp.MustCompile(pidRegexPattern).FindStringSubmatch(process)
				fmt.Println(match)
				if match != nil {
					processReal := strings.Fields(process)
					fmt.Println(processReal)
					pid, _ := strconv.Atoi(processReal[4])

					hostname, err := GetContainerHostname(pid)
					if err != nil {
						fmt.Println(err)
					}
					used, _ := strconv.Atoi(strings.TrimRight(processReal[7], "MiB"))
					gpuUsage.WithLabelValues(processReal[4], hostname).Set(float64(used))
				}
			}
			time.Sleep(time.Second * 1)
		}
	}()
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
