package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type SystemInfo struct {
	CPU struct {
		Load1  float64 `json:"load_1_min"`
		Load5  float64 `json:"load_5_min"`
		Load15 float64 `json:"load_15_min"`
	} `json:"cpu_load"`
	Memory struct {
		TotalMB     uint64  `json:"total_mb"`
		UsedMB      uint64  `json:"used_mb"`
		AvailableMB uint64  `json:"available_mb"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"memory"`

	Storage struct {
		TotalGB     float64 `json:"total_gb"`
		UsedGB      float64 `json:"used_gb"`
		FreeGB      float64 `json:"free_gb"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"storage"`

	UptimeSeconds float64 `json:"uptime_seconds"`
}

func readSystemInfo() (SystemInfo, error) {
	var info SystemInfo

	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return info, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return info, fmt.Errorf("invalid /proc/loadavg data")
	}
	info.CPU.Load1, err = strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return info, err
	}
	info.CPU.Load5, err = strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return info, err
	}
	info.CPU.Load15, err = strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return info, err
	}

	data, err = os.ReadFile("/proc/meminfo")
	if err != nil {
		return info, err
	}

	var totalKb, availableKB uint64

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			totalKb = value
		case "MemAvailable:":
			availableKB = value
		}
	}

	if totalKb == 0 {
		return info, fmt.Errorf("could not read total RAM")
	}

	usedKb := totalKb - availableKB
	info.Memory.TotalMB = totalKb / 1024
	info.Memory.UsedMB = usedKb / 1024
	info.Memory.AvailableMB = availableKB / 1024
	info.Memory.UsedPercent = float64(usedKb) / float64(totalKb) * 100

	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return info, err
	}
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	usedBytes := totalBytes - stat.Bfree*uint64(stat.Bsize)

	const gb = 1024 * 1024 * 1024

	info.Storage.TotalGB = float64(totalBytes) / float64(gb)
	info.Storage.UsedGB = float64(usedBytes) / float64(gb)
	info.Storage.FreeGB = float64(freeBytes) / float64(gb)

	if totalBytes > 0 {
		info.Storage.UsedPercent =
			float64(usedBytes) / float64(totalBytes) * 100
	}

	// Read system uptime in seconds.
	data, err = os.ReadFile("/proc/uptime")
	if err != nil {
		return info, err
	}

	fields = strings.Fields(string(data))
	if len(fields) == 0 {
		return info, fmt.Errorf("invalid /proc/uptime data")
	}

	info.UptimeSeconds, err = strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return info, err
	}

	return info, nil
}
func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET requests only", http.StatusMethodNotAllowed)
		return
	}

	info, err := readSystemInfo()
	if err != nil {
		log.Printf("reading system info: %v", err)
		http.Error(w, "Could not read system information",
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Printf("encoding response: %v", err)
	}
}

func main() {
	http.HandleFunc("/stats", statsHandler)

	log.Println("System info API listening on :8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
