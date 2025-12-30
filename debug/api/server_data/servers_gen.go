package server_data

import (
	"fmt"
	"net/http"

	datagen "github.com/go-xlite/wbx/debug/api/datagen"
	hl1 "github.com/go-xlite/wbx/utils"
	"github.com/gorilla/mux"
)

// InstanceListItem is optimized for displaying in lists
type InstanceListItem struct {
	ID          string `json:"ID"`
	Hostname    string `json:"Hostname"`
	State       string `json:"State"`
	Region      string `json:"Region"`
	Zone        string `json:"Zone"`
	LaunchedAt  string `json:"LaunchedAt"`
	Uptime      string `json:"Uptime"`
	CPUCores    int    `json:"CPUCores"`
	RAMTotalGB  int    `json:"RAMTotalGB"`
	PublicIPv4  string `json:"PublicIPv4"`
	PrivateIPv4 string `json:"PrivateIPv4"`
}

type ServerInstanceListItem struct {
	// this is to display simplified data in lists
	ID       string
	Hostname string
}

type ServerInstance struct {
	ID           string
	Hostname     string
	State        string
	LaunchedAt   string
	Uptime       string
	OS           string
	Region       string
	Zone         string
	ServerInfo   *ServerInfo
	CPUInfo      *CPUInfo
	RAMInfo      *RAMInfo
	StorageDisks []*StorageDisk
	NetworkNICs  []*NetworkNIC
}

type ServerInfo struct {
	Model           string
	Brand           string
	SerialNumber    string
	ManufactureYear int
	WarrantyExpiry  string
	Datacenter      string
	Rack            string
	Position        int
}

type CPUInfo struct {
	Brand       string
	Model       string
	Cores       int
	Threads     int
	SpeedGHz    float64
	CacheSize   int // in MB
	SocketCount int
}

type RAMInfo struct {
	TotalGB       int
	ModuleCount   int
	ModuleSizeGB  int
	Configuration string // e.g., "4x16GB", "2x64GB"
	Type          string // DDR4, DDR5
	Speed         int    // MHz
	ECC           bool
	Manufacturer  string
}

type StorageDisk struct {
	DriveID      string
	Type         string
	Brand        string
	Model        string
	SerialNumber string
	CapacityGB   int
	UsedGB       int
	AvailableGB  int
	UsagePercent float64
	Slot         int
	HealthStatus string
	TemperatureC int
}

type NetworkNIC struct {
	NICID         string
	Interface     string
	MACAddress    string
	IPv4          string
	IPv6          string
	BandwidthGbps int
	Status        string
	Driver        string
	PCISlot       string
	LinkStatus    string
	Vendor        string
	Model         string
}

type ServersDataGen struct {
	*datagen.DataGen
	instances    []*ServerInstance
	instanceList []*InstanceListItem
	listData     *ListResponse
}

// ListResponse contains column mapping and positional data
type ListResponse struct {
	Columns []string `json:"columns"`
	Data    [][]any  `json:"data"`
}

func NewServersDataGen() *ServersDataGen {
	return &ServersDataGen{
		DataGen: datagen.NewDataGen(),
	}
}

// Initialize generates all instances and prepares the optimized list
func (sdg *ServersDataGen) Initialize(count int) {
	sdg.instances = sdg.GenerateInstances(count)
	sdg.instanceList = sdg.transformToListView(sdg.instances)
	sdg.listData = sdg.transformToPositionalData(sdg.instanceList)
}

// transformToListView creates optimized list items from full instances
func (sdg *ServersDataGen) transformToListView(instances []*ServerInstance) []*InstanceListItem {
	list := make([]*InstanceListItem, 0, len(instances))
	for _, inst := range instances {
		item := &InstanceListItem{
			ID:         inst.ID,
			Hostname:   inst.Hostname,
			State:      inst.State,
			Region:     inst.Region,
			Zone:       inst.Zone,
			LaunchedAt: inst.LaunchedAt,
			Uptime:     inst.Uptime,
		}

		// Extract CPU cores
		if inst.CPUInfo != nil {
			item.CPUCores = inst.CPUInfo.Cores
		}

		// Extract RAM total
		if inst.RAMInfo != nil {
			item.RAMTotalGB = inst.RAMInfo.TotalGB
		}

		// Extract IPs from first NIC
		if len(inst.NetworkNICs) > 0 {
			item.PublicIPv4 = inst.NetworkNICs[0].IPv4
			if len(inst.NetworkNICs) > 1 {
				item.PrivateIPv4 = inst.NetworkNICs[1].IPv4
			}
		}

		list = append(list, item)
	}
	return list
}

// transformToPositionalData converts list items to positional array format
func (sdg *ServersDataGen) transformToPositionalData(list []*InstanceListItem) *ListResponse {
	columns := []string{
		"ID", "Hostname", "State", "Region", "Zone",
		"LaunchedAt", "Uptime", "CPUCores", "RAMTotalGB",
		"PublicIPv4", "PrivateIPv4",
	}

	data := make([][]any, 0, len(list))
	for _, item := range list {
		row := []any{
			item.ID,
			item.Hostname,
			item.State,
			item.Region,
			item.Zone,
			item.LaunchedAt,
			item.Uptime,
			item.CPUCores,
			item.RAMTotalGB,
			item.PublicIPv4,
			item.PrivateIPv4,
		}
		data = append(data, row)
	}

	return &ListResponse{
		Columns: columns,
		Data:    data,
	}
}

// HandleListRequest returns the optimized list view
func (sdg *ServersDataGen) HandleListRequest(w http.ResponseWriter, r *http.Request) {
	hl1.Helpers.WriteJSON(w, http.StatusOK, sdg.listData)
}

// HandleDetailsRequest returns full instance data by ID
func (sdg *ServersDataGen) HandleDetailsRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	for _, inst := range sdg.instances {
		if inst.ID == id {
			hl1.Helpers.WriteJSON(w, http.StatusOK, inst)
			return
		}
	}
	http.Error(w, "Instance not found", http.StatusNotFound)
}

// FilterOptions contains all available filter values
type FilterOptions struct {
	Regions       []string `json:"regions"`
	Zones         []string `json:"zones"`
	States        []string `json:"states"`
	InstanceTypes []string `json:"instanceTypes"`
}

// HandleFiltersRequest returns available filter options
func (sdg *ServersDataGen) HandleFiltersRequest(w http.ResponseWriter, r *http.Request) {
	regionsMap := make(map[string]bool)
	zonesMap := make(map[string]bool)
	statesMap := make(map[string]bool)
	typesMap := make(map[string]bool)

	for _, inst := range sdg.instanceList {
		if inst.Region != "" {
			regionsMap[inst.Region] = true
		}
		if inst.Zone != "" {
			zonesMap[inst.Zone] = true
		}
		if inst.State != "" {
			statesMap[inst.State] = true
		}
		// Generate instance type string
		if inst.CPUCores > 0 && inst.RAMTotalGB > 0 {
			instanceType := fmt.Sprintf("%d vCPU, %d GB RAM", inst.CPUCores, inst.RAMTotalGB)
			typesMap[instanceType] = true
		}
	}

	// Convert maps to sorted slices
	filters := FilterOptions{
		Regions:       mapKeysToSlice(regionsMap),
		Zones:         mapKeysToSlice(zonesMap),
		States:        mapKeysToSlice(statesMap),
		InstanceTypes: mapKeysToSlice(typesMap),
	}

	hl1.Helpers.WriteJSON(w, http.StatusOK, filters)
}

// Helper function to convert map keys to slice
func mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (sdg *ServersDataGen) GenerateInstances(count int) []*ServerInstance {
	records := make([]*ServerInstance, 0, count)
	for i := 0; i < count; i++ {
		records = append(records, sdg.GenerateSingleInstance(i))
	}
	return records
}

func (sdg *ServersDataGen) GenerateInstanceList(count int) []*ServerInstanceListItem {
	records := make([]*ServerInstanceListItem, 0, count)
	for i := 0; i < count; i++ {
		region := sdg.DataGen.RandomChoice(datagen.DatacenterRegions)
		zone := sdg.DataGen.RandomChoice(datagen.DatacenterZones)
		hostname := fmt.Sprintf("i-%s-%s-%04d", region, zone, sdg.DataGen.RandomInt(1000, 9999))
		id := fmt.Sprintf("i-%016x", sdg.DataGen.RandomInt(100000000, 999999999))

		records = append(records, &ServerInstanceListItem{
			ID:       id,
			Hostname: hostname,
		})
	}
	return records
}

func (sdg *ServersDataGen) GenerateSingleInstance(seed int) *ServerInstance {
	// Generate region and zone
	region := sdg.DataGen.RandomChoice(datagen.DatacenterRegions)
	zone := sdg.DataGen.RandomChoice(datagen.DatacenterZones)
	hostname := fmt.Sprintf("i-%s-%s-%04d", region, zone, sdg.DataGen.RandomInt(1000, 9999))
	id := fmt.Sprintf("i-%016x", sdg.DataGen.RandomInt(100000000, 999999999))

	// Generate launch time (random time in the past 365 days)
	daysAgo := sdg.DataGen.RandomInt(0, 365)
	launchedAt := datagen.GeneratePastDate(daysAgo)
	uptime := datagen.GenerateUptime(daysAgo)

	return &ServerInstance{
		ID:           id,
		Hostname:     hostname,
		State:        sdg.DataGen.RandomChoice(datagen.InstanceStates),
		LaunchedAt:   launchedAt,
		Uptime:       uptime,
		OS:           sdg.DataGen.RandomChoice(datagen.OSTypes),
		Region:       region,
		Zone:         zone,
		ServerInfo:   sdg.GenerateServerInfo(region, zone),
		CPUInfo:      sdg.GenerateCPUInfo(),
		RAMInfo:      sdg.GenerateRAMInfo(),
		StorageDisks: sdg.GenerateStorageDisks(),
		NetworkNICs:  sdg.GenerateNetworkNICs(hostname),
	}
}

func (sdg *ServersDataGen) GenerateServerInfo(region, zone string) *ServerInfo {
	brand := sdg.DataGen.RandomChoice(datagen.ServerBrands)
	models := datagen.ServerModels[brand]
	model := sdg.DataGen.RandomChoice(models)

	serialNumber := fmt.Sprintf("%s-%08d", brand[:2], sdg.DataGen.RandomInt(10000000, 99999999))
	manufactureYear := sdg.DataGen.RandomInt(2019, 2024)
	warrantyYears := sdg.DataGen.RandomInt(3, 5)
	warrantyExpiry := fmt.Sprintf("%d-%02d-%02d",
		manufactureYear+warrantyYears,
		sdg.DataGen.RandomInt(1, 12),
		sdg.DataGen.RandomInt(1, 28))

	datacenter := fmt.Sprintf("%s-%s", region, zone)
	rack := fmt.Sprintf("R%d", sdg.DataGen.RandomInt(1, 50))
	position := sdg.DataGen.RandomInt(1, 42)

	return &ServerInfo{
		Model:           model,
		Brand:           brand,
		SerialNumber:    serialNumber,
		ManufactureYear: manufactureYear,
		WarrantyExpiry:  warrantyExpiry,
		Datacenter:      datacenter,
		Rack:            rack,
		Position:        position,
	}
}

func (sdg *ServersDataGen) GenerateCPUInfo() *CPUInfo {
	brand := sdg.DataGen.RandomChoice(datagen.CPUBrands)
	models := datagen.CPUModels[brand]
	model := sdg.DataGen.RandomChoice(models)

	var cores, cacheSize int
	var speedGHz float64

	if brand == "Intel" {
		cores = sdg.RandomIntChoice([]int{8, 12, 16, 20, 24, 28, 32})
		speedGHz = sdg.DataGen.RandomFloat(2.0, 3.8)
		cacheSize = cores * sdg.DataGen.RandomInt(1, 2) // MB per core roughly
	} else { // AMD
		cores = sdg.RandomIntChoice([]int{8, 16, 24, 32, 48, 64})
		speedGHz = sdg.DataGen.RandomFloat(2.2, 3.5)
		cacheSize = cores * sdg.DataGen.RandomInt(2, 4)
	}

	threads := cores * 2 // Assuming hyperthreading/SMT
	socketCount := sdg.RandomIntChoice([]int{1, 2, 4})

	return &CPUInfo{
		Brand:       brand,
		Model:       model,
		Cores:       cores * socketCount,
		Threads:     threads * socketCount,
		SpeedGHz:    speedGHz,
		CacheSize:   cacheSize * socketCount,
		SocketCount: socketCount,
	}
}

func (sdg *ServersDataGen) GenerateRAMInfo() *RAMInfo {
	// Common RAM configurations for servers
	configs := []struct {
		moduleCount  int
		moduleSizeGB int
	}{
		{4, 8},   // 32GB total
		{4, 16},  // 64GB total
		{4, 32},  // 128GB total
		{8, 16},  // 128GB total
		{8, 32},  // 256GB total
		{8, 64},  // 512GB total
		{16, 32}, // 512GB total
		{16, 64}, // 1024GB (1TB) total
		{12, 32}, // 384GB total
		{12, 64}, // 768GB total
		{2, 64},  // 128GB total
		{2, 128}, // 256GB total
		{4, 128}, // 512GB total
	}

	config := configs[sdg.DataGen.RandomInt(0, len(configs)-1)]
	moduleCount := config.moduleCount
	moduleSizeGB := config.moduleSizeGB
	totalGB := moduleCount * moduleSizeGB

	// Generate configuration string
	configStr := fmt.Sprintf("%dx%dGB", moduleCount, moduleSizeGB)

	// RAM type - DDR4 is more common, DDR5 for newer servers
	ramType := sdg.DataGen.RandomChoice([]string{"DDR4", "DDR4", "DDR4", "DDR5"})

	// Speed varies by type
	var speed int
	if ramType == "DDR5" {
		speed = sdg.RandomIntChoice([]int{4800, 5200, 5600, 6000, 6400})
	} else {
		speed = sdg.RandomIntChoice([]int{2133, 2400, 2666, 2933, 3200})
	}

	// ECC is standard for servers
	ecc := true

	manufacturer := sdg.DataGen.RandomChoice([]string{"Samsung", "Micron", "SK Hynix", "Kingston", "Crucial", "Corsair"})

	return &RAMInfo{
		TotalGB:       totalGB,
		ModuleCount:   moduleCount,
		ModuleSizeGB:  moduleSizeGB,
		Configuration: configStr,
		Type:          ramType,
		Speed:         speed,
		ECC:           ecc,
		Manufacturer:  manufacturer,
	}
}

func (sdg *ServersDataGen) GenerateStorageDisks() []*StorageDisk {
	// Decide how many drives to generate (2-12 drives)
	driveCount := sdg.RandomIntChoice([]int{2, 4, 6, 8, 10, 12})
	disks := make([]*StorageDisk, 0, driveCount)

	// Pick primary storage type for this server
	storageType := sdg.DataGen.RandomChoice(datagen.StorageTypes)

	for i := 0; i < driveCount; i++ {
		brand := sdg.DataGen.RandomChoice(datagen.StorageBrands)

		var capacityGB int
		var model string

		switch storageType {
		case "NVMe SSD":
			capacityGB = sdg.RandomIntChoice([]int{960, 1920, 3840, 7680})
			model = fmt.Sprintf("%s NVMe %dGB", brand, capacityGB)
		case "SATA SSD", "SAS SSD":
			capacityGB = sdg.RandomIntChoice([]int{480, 960, 1920, 3840})
			model = fmt.Sprintf("%s %s %dGB", brand, storageType, capacityGB)
		case "SAS HDD":
			capacityGB = sdg.RandomIntChoice([]int{2000, 4000, 8000, 12000})
			model = fmt.Sprintf("%s SAS HDD %dGB", brand, capacityGB)
		}

		usagePercent := sdg.DataGen.RandomFloat(15.0, 85.0)
		usedGB := int(float64(capacityGB) * usagePercent / 100.0)
		availableGB := capacityGB - usedGB

		serialNumber := fmt.Sprintf("%s%d%08d", brand[:min(3, len(brand))], i, sdg.DataGen.RandomInt(10000000, 99999999))
		driveID := fmt.Sprintf("disk-%d", i)

		healthStatus := sdg.DataGen.RandomChoice([]string{"healthy", "healthy", "healthy", "healthy", "warning"})
		temperature := sdg.DataGen.RandomInt(28, 55)

		disks = append(disks, &StorageDisk{
			DriveID:      driveID,
			Type:         storageType,
			Brand:        brand,
			Model:        model,
			SerialNumber: serialNumber,
			CapacityGB:   capacityGB,
			UsedGB:       usedGB,
			AvailableGB:  availableGB,
			UsagePercent: usagePercent,
			Slot:         i,
			HealthStatus: healthStatus,
			TemperatureC: temperature,
		})
	}

	return disks
}

func (sdg *ServersDataGen) GenerateNetworkNICs(hostname string) []*NetworkNIC {
	// Generate 2-4 network interfaces
	nicCount := sdg.RandomIntChoice([]int{2, 2, 4, 4})
	nics := make([]*NetworkNIC, 0, nicCount)

	nicVendors := []string{"Intel", "Broadcom", "Mellanox", "Realtek"}
	nicModels := map[string][]string{
		"Intel":    {"I350 Gigabit", "X710 10GbE", "XXV710 25GbE", "E810 100GbE"},
		"Broadcom": {"NetXtreme BCM5720", "NetXtreme E-Series BCM57414", "NetXtreme-E BCM57508"},
		"Mellanox": {"ConnectX-4", "ConnectX-5", "ConnectX-6"},
		"Realtek":  {"RTL8111/8168/8411", "RTL8125 2.5GbE"},
	}

	interfaceNames := []string{"eth", "ens", "eno", "enp"}

	for i := 0; i < nicCount; i++ {
		vendor := sdg.DataGen.RandomChoice(nicVendors)
		models := nicModels[vendor]
		model := sdg.DataGen.RandomChoice(models)

		// Determine bandwidth based on model
		var bandwidth int
		if contains(model, "100GbE") || contains(model, "ConnectX-6") {
			bandwidth = 100
		} else if contains(model, "25GbE") || contains(model, "ConnectX-5") {
			bandwidth = 25
		} else if contains(model, "10GbE") || contains(model, "ConnectX-4") {
			bandwidth = 10
		} else if contains(model, "2.5GbE") {
			bandwidth = 2
		} else {
			bandwidth = 1
		}

		// Generate interface name
		ifacePrefix := sdg.DataGen.RandomChoice(interfaceNames)
		iface := fmt.Sprintf("%s%d", ifacePrefix, i)

		// Generate MAC address
		mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			sdg.DataGen.RandomInt(0, 255),
			sdg.DataGen.RandomInt(0, 255),
			sdg.DataGen.RandomInt(0, 255),
			sdg.DataGen.RandomInt(0, 255),
			sdg.DataGen.RandomInt(0, 255),
			sdg.DataGen.RandomInt(0, 255))

		// First NIC gets public IP, others get private
		var ipv4 string
		if i == 0 {
			ipv4 = fmt.Sprintf("%d.%d.%d.%d",
				sdg.RandomIntChoice([]int{3, 18, 34, 52, 54}),
				sdg.DataGen.RandomInt(0, 255),
				sdg.DataGen.RandomInt(0, 255),
				sdg.DataGen.RandomInt(1, 254))
		} else {
			ipv4 = fmt.Sprintf("10.%d.%d.%d",
				sdg.DataGen.RandomInt(0, 255),
				sdg.DataGen.RandomInt(0, 255),
				sdg.DataGen.RandomInt(1, 254))
		}

		ipv6 := fmt.Sprintf("2600:1f18:%04x:%04x::%x",
			sdg.DataGen.RandomInt(0, 9999),
			sdg.DataGen.RandomInt(0, 9999),
			sdg.DataGen.RandomInt(0, 9999))

		status := sdg.DataGen.RandomChoice([]string{"up", "up", "up", "down"})
		linkStatus := "connected"
		if status == "down" {
			linkStatus = "disconnected"
		}

		pciSlot := fmt.Sprintf("%02x:%02x.%d",
			sdg.DataGen.RandomInt(0, 255),
			sdg.DataGen.RandomInt(0, 31),
			sdg.DataGen.RandomInt(0, 7))

		driver := sdg.DataGen.RandomChoice([]string{"igb", "ixgbe", "i40e", "ice", "bnxt_en", "mlx5_core"})

		nics = append(nics, &NetworkNIC{
			NICID:         fmt.Sprintf("nic-%d", i),
			Interface:     iface,
			MACAddress:    mac,
			IPv4:          ipv4,
			IPv6:          ipv6,
			BandwidthGbps: bandwidth,
			Status:        status,
			Driver:        driver,
			PCISlot:       pciSlot,
			LinkStatus:    linkStatus,
			Vendor:        vendor,
			Model:         model,
		})
	}

	return nics
}

func (sdg *ServersDataGen) RandomIntChoice(items []int) int {
	if len(items) == 0 {
		return 0
	}
	return items[sdg.DataGen.RandomInt(0, len(items)-1)]
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
