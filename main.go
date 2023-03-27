package main

import (
    "fmt"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
    gpuMemRegexPattern = `(\d+)MiB \/ (\d+)MiB`
    pidRegexPattern    = `(\d+)\.h264_videorecorder`
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

func init() {
    prometheus.MustRegister(gpuUsage)
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

            processes := strings.Split(string(out), "|")
            //fmt.Println(processes)

            for _, process := range processes[1:] {
                fmt.Println(process)
                match := regexp.MustCompile(pidRegexPattern).FindStringSubmatch(process)
                fmt.Println(match)
                if match != nil {
                    pid, _ := strconv.Atoi(match[1])
                    dockerHostnameCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("ancestor=%d.h264_videorecorder", pid), "--format", "{{.Names}}")
                    hostnameBytes, err := dockerHostnameCmd.Output()

                    if err != nil {
                        fmt.Println(err)
                    }

                    hostname := strings.TrimSpace(string(hostnameBytes))
                    gpuMemMatch := regexp.MustCompile(gpuMemRegexPattern).FindStringSubmatch(process)

                    if gpuMemMatch != nil {
                        used, _ := strconv.Atoi(gpuMemMatch[1])
                        total, _ := strconv.Atoi(gpuMemMatch[2])
                        usagePercentage := float64(used) / float64(total) * 100
                        gpuUsage.WithLabelValues(match[1], hostname).Set(usagePercentage)
                    }
                }
            }
        }
    }()
    http.ListenAndServe(":8080", nil)
}
