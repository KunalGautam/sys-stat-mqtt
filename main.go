package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemDetails defines hardware specs of the host
type SystemDetails struct {
	CPUModel      string  `json:"cpu_model"`
	CPULogical    int     `json:"cpu_logical_cores"`
	CPUPhysical   int     `json:"cpu_physical_cores"`
	TotalMemoryGB float64 `json:"total_memory_gb"`
	TotalDiskGB   float64 `json:"total_disk_gb"`
}

// SystemStats defines the structure of our JSON payload
type SystemStats struct {
	CPUPercent    float64       `json:"cpu_percent"`
	MemoryPercent float64       `json:"memory_percent"`
	DiskPercent   float64       `json:"disk_percent"`
	CPUTemp       float64       `json:"cpu_temp_c"`
	DiskTemp      float64       `json:"disk_temp_c"`
	Load1         float64       `json:"load_1m"`
	Load5         float64       `json:"load_5m"`
	Load15        float64       `json:"load_15m"`
	Details       SystemDetails `json:"details"`
}

func main() {
	// Load .env file (ignore error if it doesn't exist, as env vars might be set directly in the environment)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://localhost:1883"
	}

	clientID := os.Getenv("MQTT_CLIENT_ID")
	if clientID == "" {
		clientID = "mac-linux-sysmon"
	}

	topic := os.Getenv("MQTT_TOPIC") + clientID
	if topic == "" {
		topic = "system/stats" + clientID
	}

	intervalStr := os.Getenv("STATS_INTERVAL")
	interval := 5 * time.Second
	if intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil {
			interval = d
		} else {
			log.Printf("Invalid STATS_INTERVAL '%s', using default 5s: %v", intervalStr, err)
		}
	}

	username := os.Getenv("MQTT_USERNAME")
	password := os.Getenv("MQTT_PASSWORD")

	// 1. Configure MQTT Client Options
	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientID)
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetAutoReconnect(true) // Handles network drops automatically

	// 2. Connect to the Broker
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error connecting to MQTT broker: %v", token.Error())
	}
	defer client.Disconnect(250)
	fmt.Printf("Connected to MQTT Broker (%s). Publishing to '%s'...\n", broker, topic)

	// 3. Set up a Ticker for regular intervals
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 4. Handle OS interrupts gracefully (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main loop
	for {
		select {
		case <-ticker.C:
			// Fetch stats
			stats, err := getSystemStats()
			if err != nil {
				log.Printf("Error gathering stats: %v", err)
				continue
			}

			// Marshal to JSON
			payload, err := json.Marshal(stats)
			if err != nil {
				log.Printf("Error marshaling JSON: %v", err)
				continue
			}

			// Publish to MQTT (QoS 1 ensures delivery)
			token := client.Publish(topic, 1, false, payload)
			token.Wait() // Wait for broker acknowledgment

			log.Printf("Published stats: CPU: %.1f%% (%.1f°C), Mem: %.1f%%, Disk: %.1f%% (%.1f°C), Load: %.2f %.2f %.2f",
				stats.CPUPercent, stats.CPUTemp, stats.MemoryPercent, stats.DiskPercent, stats.DiskTemp, stats.Load1, stats.Load5, stats.Load15)

		case <-sigChan:
			fmt.Println("\nStopping system monitor daemon...")
			return
		}
	}
}

// getSystemStats gathers platform-agnostic metrics
func getSystemStats() (SystemStats, error) {
	var stats SystemStats

	// CPU usage over the last second
	cpuPercents, err := cpu.Percent(time.Second, false)
	if err != nil || len(cpuPercents) == 0 {
		return stats, fmt.Errorf("failed to get CPU stats: %w", err)
	}
	stats.CPUPercent = round2(cpuPercents[0])

	// Virtual Memory usage
	vMem, err := mem.VirtualMemory()
	if err != nil {
		return stats, fmt.Errorf("failed to get Memory stats: %w", err)
	}
	stats.MemoryPercent = round2(vMem.UsedPercent)

	// Disk usage (Root directory "/" works perfectly on both macOS and Linux)
	diskUsage, err := disk.Usage("/")
	if err != nil {
		return stats, fmt.Errorf("failed to get Disk stats: %w", err)
	}
	stats.DiskPercent = round2(diskUsage.UsedPercent)

	// Temperatures
	temps, err := host.SensorsTemperatures()
	if err == nil {
		for _, t := range temps {
			key := strings.ToLower(t.SensorKey)
			// CPU Temp: prefers package_id_0, coretemp, or anything containing cpu/coretemp
			if t.SensorKey == "coretemp_package_id_0" {
				stats.CPUTemp = round2(t.Temperature)
			} else if stats.CPUTemp == 0 && (strings.Contains(key, "cpu") || strings.Contains(key, "coretemp")) {
				stats.CPUTemp = round2(t.Temperature)
			}

			// Disk Temp: prefers nvme_composite, or anything containing nvme, sda, sdb, disk, hdd
			if t.SensorKey == "nvme_composite" {
				stats.DiskTemp = round2(t.Temperature)
			} else if stats.DiskTemp == 0 && (strings.Contains(key, "nvme") || strings.Contains(key, "sda") || strings.Contains(key, "sdb") || strings.Contains(key, "disk") || strings.Contains(key, "hdd")) {
				stats.DiskTemp = round2(t.Temperature)
			}
		}
	}

	// Load Averages
	avg, err := load.Avg()
	if err == nil {
		stats.Load1 = round2(avg.Load1)
		stats.Load5 = round2(avg.Load5)
		stats.Load15 = round2(avg.Load15)
	}

	// Details (Hardware Specifications)
	var cpuModel string
	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		cpuModel = info[0].ModelName
	}
	logicalCores, _ := cpu.Counts(true)
	physicalCores, _ := cpu.Counts(false)

	stats.Details = SystemDetails{
		CPUModel:      cpuModel,
		CPULogical:    logicalCores,
		CPUPhysical:   physicalCores,
		TotalMemoryGB: round2(float64(vMem.Total) / (1024 * 1024 * 1024)),
		TotalDiskGB:   round2(float64(diskUsage.Total) / (1024 * 1024 * 1024)),
	}

	return stats, nil
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}
