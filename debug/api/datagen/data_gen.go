package datagen

import (
	"fmt"
	"math/rand"
	"time"
)

type DataGen struct {
	rand *rand.Rand
}

func NewDataGen() *DataGen {
	return &DataGen{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Random selection helpers
func (dg *DataGen) RandomChoice(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[dg.rand.Intn(len(items))]
}

func (dg *DataGen) RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return min + dg.rand.Intn(max-min+1)
}

func (dg *DataGen) RandomFloat(min, max float64) float64 {
	return min + dg.rand.Float64()*(max-min)
}

func (dg *DataGen) RandomBool() bool {
	return dg.rand.Intn(2) == 1
}

// Data pools for realistic generation
var (
	ServerBrands = []string{"Dell", "HP", "Supermicro", "Lenovo", "Cisco", "IBM", "Fujitsu"}
	ServerModels = map[string][]string{
		"Dell":       {"PowerEdge R750", "PowerEdge R650", "PowerEdge R7525", "PowerEdge R6525"},
		"HP":         {"ProLiant DL380 Gen10", "ProLiant DL360 Gen10", "ProLiant DL385 Gen10"},
		"Supermicro": {"SYS-2029U-TN24R4T", "SYS-1029U-TRTP", "SYS-2029P-C1RT"},
		"Lenovo":     {"ThinkSystem SR650", "ThinkSystem SR630", "ThinkSystem SR850"},
		"Cisco":      {"UCS C240 M6", "UCS C220 M6", "UCS C480 M5"},
		"IBM":        {"System x3650 M5", "Power System S922", "System x3850 X6"},
		"Fujitsu":    {"PRIMERGY RX2540 M5", "PRIMERGY RX2530 M5", "PRIMERGY RX4770 M5"},
	}

	CPUBrands = []string{"Intel", "AMD"}
	CPUModels = map[string][]string{
		"Intel": {
			"Xeon Gold 6248R", "Xeon Gold 6230R", "Xeon Platinum 8280",
			"Xeon Silver 4214", "Xeon Gold 5220R", "Xeon Platinum 8380",
		},
		"AMD": {
			"EPYC 7763", "EPYC 7713", "EPYC 7543",
			"EPYC 7443", "EPYC 7313", "EPYC 7742",
		},
	}

	StorageTypes  = []string{"NVMe SSD", "SATA SSD", "SAS SSD", "SAS HDD"}
	StorageBrands = []string{"Samsung", "Intel", "Micron", "Western Digital", "Seagate", "Toshiba"}

	OSTypes = []string{"Ubuntu 22.04 LTS", "Ubuntu 20.04 LTS", "CentOS 8", "RHEL 8.5", "Debian 11", "Windows Server 2022", "Windows Server 2019"}

	DatacenterRegions = []string{"us-east-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-southeast-1", "ap-northeast-1", "ca-central-1"}
	DatacenterZones   = []string{"a", "b", "c", "d"}

	InstanceStates = []string{"running", "stopped", "pending", "stopping", "terminated"}
)

// GeneratePastDate generates a date string for a date in the past
func GeneratePastDate(daysAgo int) string {
	pastTime := time.Now().AddDate(0, 0, -daysAgo)
	return pastTime.Format("2006-01-02 15:04:05")
}

// GenerateUptime generates a human-readable uptime string
func GenerateUptime(days int) string {
	if days == 0 {
		return "< 1 day"
	}
	if days < 30 {
		return fmt.Sprintf("%d days", days)
	}
	months := days / 30
	remainingDays := days % 30
	if remainingDays == 0 {
		return fmt.Sprintf("%d months", months)
	}
	return fmt.Sprintf("%d months, %d days", months, remainingDays)
}
