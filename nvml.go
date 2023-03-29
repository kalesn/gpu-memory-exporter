package main

import (
	"fmt"
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
)

type ProcessesInfo struct {
	Pid  int
	Used int
}

var processesInfos []ProcessesInfo

func GetAllRunningProcesses() error {
	// Initialize NVML library
	err := nvml.Init()
	defer func() {
		_ = nvml.Shutdown()
	}()

	if err != nil {
		fmt.Println("Failed to initialize NVML:", err)
		return err
	}

	// Get the number of GPUs in the system
	count, err := nvml.GetDeviceCount()
	if err != nil {
		fmt.Println("Failed to get GPU count:", err)
		return err
	}

	for i := uint(0); i < count; i++ {
		// Get GPU handle
		gpu, err := nvml.NewDeviceLite(i)
		if err != nil {
			fmt.Printf("Failed to get handle for GPU %d: %v\n", i, err)
			continue
		}

		// Get list of processes running on this GPU
		processes, err := gpu.GetAllRunningProcesses()
		if err != nil {
			fmt.Printf("Failed to get processes for GPU %d: %v\n", i, err)
			continue
		}

		fmt.Printf("GPU %d processes:\n", i)
		for _, process := range processes {
			ProcessesInfo := ProcessesInfo{
				Pid:  int(process.PID),
				Used: int(process.MemoryUsed),
			}
			processesInfos = append(processesInfos, ProcessesInfo)
			fmt.Printf("\tProcess name: %s, PID: %d, Used memory: %d MB\n",
				process.Name, process.PID, process.MemoryUsed)
		}
	}
	return nil
}
