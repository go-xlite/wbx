package main

import (
	"encoding/json"
	"fmt"
	"log"

	server_data "github.com/go-xlite/wbx/debug/api/server_data"
)

func main() {
	// Create a new server data generator
	gen := server_data.NewServersDataGen()

	// Example 1: Generate a list of server instances (lightweight)
	fmt.Println("=== Example 1: Server Instance List ===")
	instanceList := gen.GenerateInstanceList(5)
	for i, instance := range instanceList {
		fmt.Printf("%d. ID: %s, Hostname: %s\n", i+1, instance.ID, instance.Hostname)
	}
	fmt.Println()

	// Example 2: Generate full server instances with all details
	fmt.Println("=== Example 2: Full Server Instances ===")
	instances := gen.GenerateInstances(3)

	for i, instance := range instances {
		fmt.Printf("\n--- Instance %d ---\n", i+1)
		fmt.Printf("ID: %s\n", instance.ID)
		fmt.Printf("Hostname: %s\n", instance.Hostname)
		fmt.Printf("State: %s\n", instance.State)
		fmt.Printf("OS: %s\n", instance.OS)
		fmt.Printf("Region: %s, Zone: %s\n", instance.Region, instance.Zone)
		fmt.Printf("Launched: %s\n", instance.LaunchedAt)
		fmt.Printf("Uptime: %s\n", instance.Uptime)

		if instance.ServerInfo != nil {
			fmt.Printf("\nServer Hardware:\n")
			fmt.Printf("  Brand: %s\n", instance.ServerInfo.Brand)
			fmt.Printf("  Model: %s\n", instance.ServerInfo.Model)
			fmt.Printf("  Serial: %s\n", instance.ServerInfo.SerialNumber)
			fmt.Printf("  Location: %s, Rack %s, Position %d\n",
				instance.ServerInfo.Datacenter,
				instance.ServerInfo.Rack,
				instance.ServerInfo.Position)
		}

		if instance.CPUInfo != nil {
			fmt.Printf("\nCPU Info:\n")
			fmt.Printf("  %s %s\n", instance.CPUInfo.Brand, instance.CPUInfo.Model)
			fmt.Printf("  Cores: %d, Threads: %d\n", instance.CPUInfo.Cores, instance.CPUInfo.Threads)
			fmt.Printf("  Speed: %.2f GHz, Cache: %d MB\n", instance.CPUInfo.SpeedGHz, instance.CPUInfo.CacheSize)
			fmt.Printf("  Sockets: %d\n", instance.CPUInfo.SocketCount)
		}

		if instance.RAMInfo != nil {
			fmt.Printf("\nRAM Info:\n")
			fmt.Printf("  Configuration: %s (%d GB total)\n", instance.RAMInfo.Configuration, instance.RAMInfo.TotalGB)
			fmt.Printf("  Type: %s %d MHz\n", instance.RAMInfo.Type, instance.RAMInfo.Speed)
			fmt.Printf("  Manufacturer: %s\n", instance.RAMInfo.Manufacturer)
			eccStatus := "Non-ECC"
			if instance.RAMInfo.ECC {
				eccStatus = "ECC"
			}
			fmt.Printf("  ECC: %s\n", eccStatus)
		}

		if len(instance.StorageDisks) > 0 {
			fmt.Printf("\nStorage Disks (%d drives):\n", len(instance.StorageDisks))
			totalCapacity := 0
			totalUsed := 0
			for _, disk := range instance.StorageDisks {
				fmt.Printf("  [%s] Slot %d: %s %s\n", disk.DriveID, disk.Slot, disk.Brand, disk.Model)
				fmt.Printf("    Capacity: %d GB, Used: %d GB (%.1f%%), Health: %s, Temp: %d°C\n",
					disk.CapacityGB, disk.UsedGB, disk.UsagePercent, disk.HealthStatus, disk.TemperatureC)
				totalCapacity += disk.CapacityGB
				totalUsed += disk.UsedGB
			}
			fmt.Printf("  Total Capacity: %d GB, Total Used: %d GB\n", totalCapacity, totalUsed)
		}

		if len(instance.NetworkNICs) > 0 {
			fmt.Printf("\nNetwork Interfaces (%d NICs):\n", len(instance.NetworkNICs))
			for _, nic := range instance.NetworkNICs {
				fmt.Printf("  [%s] %s: %s %s\n", nic.NICID, nic.Interface, nic.Vendor, nic.Model)
				fmt.Printf("    IPv4: %s, IPv6: %s\n", nic.IPv4, nic.IPv6)
				fmt.Printf("    MAC: %s, Bandwidth: %d Gbps\n", nic.MACAddress, nic.BandwidthGbps)
				fmt.Printf("    Status: %s, Link: %s, PCI: %s, Driver: %s\n",
					nic.Status, nic.LinkStatus, nic.PCISlot, nic.Driver)
			}
		}
	}

	// Example 3: Generate JSON for API response
	fmt.Println("\n\n=== Example 3: JSON Output ===")
	jsonData, err := json.MarshalIndent(instances[0], "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonData))
}
