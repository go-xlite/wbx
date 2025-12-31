import { pathPrefix } from '../../../lib/site_core.js';
import { createAPIManager } from './api-manager.js';

let apiManager = null;
let servers = [];
let filteredServers = [];
let currentPage = 1;
const itemsPerPage = 20;
let filterOptions = null;
let activeFilters = {
    search: '',
    region: '',
    zone: '',
    state: '',
    type: ''
};
let columnMapping = {}; // Maps column names to array indices

function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// Convert positional array to object using column mapping
function arrayToObject(arr) {
    const obj = {};
    for (const key in columnMapping) {
        obj[key] = arr[columnMapping[key]];
    }
    return obj;
}

function updateStats() {
    document.getElementById('totalServers').textContent = servers.length;
    document.getElementById('displayedServers').textContent = filteredServers.length;
}

function parseHostname(hostname) {
    // Parse hostname like "i-us-east-1-a-5761" to extract region and zone
    const parts = hostname.split('-');
    if (parts.length >= 4) {
        const region = `${parts[1]}-${parts[2]}-${parts[3]}`;
        const zone = parts[4];
        return { region, zone };
    }
    return { region: '-', zone: '-' };
}

function getStatusBadge(state) {
    const stateMap = {
        'running': '<span class="status-badge status-running">● Running</span>',
        'stopped': '<span class="status-badge status-stopped">● Stopped</span>',
        'pending': '<span class="status-badge status-pending">● Pending</span>',
        'stopping': '<span class="status-badge status-stopping">● Stopping</span>',
        'terminated': '<span class="status-badge status-terminated">● Terminated</span>'
    };
    return stateMap[state] || `<span class="status-badge">${escapeHtml(state)}</span>`;
}

function getInstanceType(server) {
    if (!server.CPUCores || !server.RAMTotalGB) return '-';
    return `${server.CPUCores} vCPU, ${server.RAMTotalGB} GB RAM`;
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function renderServerRow(server) {
    // Server can be either an object (for filtered view) or need to be accessed via columnMapping
    const getVal = (key) => typeof server === 'object' && !Array.isArray(server) 
        ? server[key] 
        : server[columnMapping[key]];
    
    return `
        <tr class="clickable-row" data-instance-id="${escapeHtml(getVal('ID'))}">
            <td><span class="instance-id">${escapeHtml(getVal('ID'))}</span></td>
            <td><span class="instance-name">${escapeHtml(getVal('Hostname'))}</span></td>
            <td>${getStatusBadge(getVal('State'))}</td>
            <td><span class="instance-type">${getVal('CPUCores')} vCPU, ${getVal('RAMTotalGB')} GB RAM</span></td>
            <td><span class="instance-ip">${escapeHtml(getVal('PublicIPv4') || '-')}</span></td>
            <td><span class="instance-region">${escapeHtml(getVal('Region') || '-')}</span></td>
            <td><span class="instance-date">${formatDate(getVal('LaunchedAt'))}</span></td>
        </tr>
    `;
}

function renderPagination() {
    const totalPages = Math.ceil(filteredServers.length / itemsPerPage);
    const paginationDiv = document.getElementById('pagination');
    
    if (totalPages <= 1) {
        paginationDiv.innerHTML = '';
        return;
    }

    let html = '';
    
    // Previous button
    if (currentPage > 1) {
        html += '<button onclick="window.goToPage(' + (currentPage - 1) + ')">← Previous</button>';
    }
    
    // Page numbers
    const maxVisiblePages = 7;
    let startPage = Math.max(1, currentPage - Math.floor(maxVisiblePages / 2));
    let endPage = Math.min(totalPages, startPage + maxVisiblePages - 1);
    
    if (endPage - startPage < maxVisiblePages - 1) {
        startPage = Math.max(1, endPage - maxVisiblePages + 1);
    }
    
    if (startPage > 1) {
        html += '<button onclick="window.goToPage(1)">1</button>';
        if (startPage > 2) html += '<span>...</span>';
    }
    
    for (let i = startPage; i <= endPage; i++) {
        const active = i === currentPage ? 'active' : '';
        html += '<button class="' + active + '" onclick="window.goToPage(' + i + ')">' + i + '</button>';
    }
    
    if (endPage < totalPages) {
        if (endPage < totalPages - 1) html += '<span>...</span>';
        html += '<button onclick="window.goToPage(' + totalPages + ')">' + totalPages + '</button>';
    }
    
    // Next button
    if (currentPage < totalPages) {
        html += '<button onclick="window.goToPage(' + (currentPage + 1) + ')">Next →</button>';
    }
    
    paginationDiv.innerHTML = html;
}

function renderServers() {
    const tbody = document.getElementById('instancesTableBody');
    const start = (currentPage - 1) * itemsPerPage;
    const end = start + itemsPerPage;
    const pageServers = filteredServers.slice(start, end);
    
    if (pageServers.length === 0) {
        tbody.innerHTML = '<tr class="empty-row"><td colspan="7" class="empty-state">No instances found</td></tr>';
        return;
    }
    
    tbody.innerHTML = pageServers.map(renderServerRow).join('');
    renderPagination();
    attachRowClickHandlers();
}

function filterServers(searchTerm) {
    activeFilters.search = searchTerm || '';
    applyFilters();
}

function applyFilters() {
    filteredServers = servers.filter(server => {
        // Convert array to object if needed
        const serverObj = Array.isArray(server) ? arrayToObject(server) : server;
        
        // Search filter
        if (activeFilters.search) {
            const term = activeFilters.search.toLowerCase();
            const matchesSearch = 
                serverObj.ID.toLowerCase().includes(term) ||
                serverObj.Hostname.toLowerCase().includes(term) ||
                (serverObj.PublicIPv4 && serverObj.PublicIPv4.toLowerCase().includes(term)) ||
                (serverObj.PrivateIPv4 && serverObj.PrivateIPv4.toLowerCase().includes(term));
            
            if (!matchesSearch) return false;
        }

        // Region filter
        if (activeFilters.region && serverObj.Region !== activeFilters.region) {
            return false;
        }

        // Zone filter
        if (activeFilters.zone && serverObj.Zone !== activeFilters.zone) {
            return false;
        }

        // State filter
        if (activeFilters.state && serverObj.State !== activeFilters.state) {
            return false;
        }

        // Instance type filter
        if (activeFilters.type) {
            const instanceType = getInstanceType(serverObj);
            if (instanceType !== activeFilters.type) {
                return false;
            }
        }

        return true;
    });

    currentPage = 1;
    updateStats();
    renderServers();
}

function populateFilterDropdowns(filters) {
    filterOptions = filters;

    // Populate regions
    const regionSelect = document.getElementById('regionFilter');
    filters.regions.forEach(region => {
        const option = document.createElement('option');
        option.value = region;
        option.textContent = region;
        regionSelect.appendChild(option);
    });

    // Populate zones
    const zoneSelect = document.getElementById('zoneFilter');
    filters.zones.forEach(zone => {
        const option = document.createElement('option');
        option.value = zone;
        option.textContent = zone;
        zoneSelect.appendChild(option);
    });

    // Populate states
    const stateSelect = document.getElementById('stateFilter');
    filters.states.forEach(state => {
        const option = document.createElement('option');
        option.value = state;
        option.textContent = state.charAt(0).toUpperCase() + state.slice(1);
        stateSelect.appendChild(option);
    });

    // Populate instance types
    const typeSelect = document.getElementById('typeFilter');
    filters.instanceTypes.forEach(type => {
        const option = document.createElement('option');
        option.value = type;
        option.textContent = type;
        typeSelect.appendChild(option);
    });
}

function attachRowClickHandlers() {
    const rows = document.querySelectorAll('.clickable-row');
    rows.forEach(row => {
        row.addEventListener('click', async () => {
            const instanceId = row.getAttribute('data-instance-id');
            await showInstanceDetails(instanceId);
        });
    });
}

async function showInstanceDetails(instanceId) {
    const modal = document.getElementById('detailsModal');
    const modalContent = document.getElementById('modalContent');
    
    modal.style.display = 'flex';
    modalContent.innerHTML = '<div class="modal-loader">Loading instance details...</div>';
    
    try {
        const details = await apiManager.fetchServerDetails(instanceId);
        modalContent.innerHTML = renderInstanceDetails(details);
    } catch (error) {
        modalContent.innerHTML = `<div class="error-state">Failed to load details: ${escapeHtml(error.message)}</div>`;
    }
}

function renderInstanceDetails(instance) {
    let html = `
        <div class="details-header">
            <h2>${escapeHtml(instance.Hostname)}</h2>
            <div class="details-id">${escapeHtml(instance.ID)}</div>
        </div>
        
        <div class="details-section">
            <h3>Instance Information</h3>
            <div class="details-grid">
                <div class="detail-item"><strong>Status:</strong> ${getStatusBadge(instance.State)}</div>
                <div class="detail-item"><strong>Region:</strong> ${escapeHtml(instance.Region || '-')}</div>
                <div class="detail-item"><strong>Zone:</strong> ${escapeHtml(instance.Zone || '-')}</div>
                <div class="detail-item"><strong>OS:</strong> ${escapeHtml(instance.OS || '-')}</div>
                <div class="detail-item"><strong>Launched:</strong> ${formatDate(instance.LaunchedAt)}</div>
                <div class="detail-item"><strong>Uptime:</strong> ${escapeHtml(instance.Uptime || '-')}</div>
            </div>
        </div>
    `;
    
    if (instance.ServerInfo) {
        html += `
            <div class="details-section">
                <h3>Server Hardware</h3>
                <div class="details-grid">
                    <div class="detail-item"><strong>Brand:</strong> ${escapeHtml(instance.ServerInfo.Brand)}</div>
                    <div class="detail-item"><strong>Model:</strong> ${escapeHtml(instance.ServerInfo.Model)}</div>
                    <div class="detail-item"><strong>Serial Number:</strong> ${escapeHtml(instance.ServerInfo.SerialNumber)}</div>
                    <div class="detail-item"><strong>Manufacture Year:</strong> ${instance.ServerInfo.ManufactureYear}</div>
                    <div class="detail-item"><strong>Warranty Expiry:</strong> ${escapeHtml(instance.ServerInfo.WarrantyExpiry)}</div>
                    <div class="detail-item"><strong>Datacenter:</strong> ${escapeHtml(instance.ServerInfo.Datacenter)}</div>
                    <div class="detail-item"><strong>Rack:</strong> ${escapeHtml(instance.ServerInfo.Rack)}</div>
                    <div class="detail-item"><strong>Position:</strong> ${instance.ServerInfo.Position}</div>
                </div>
            </div>
        `;
    }
    
    if (instance.CPUInfo) {
        html += `
            <div class="details-section">
                <h3>CPU Information</h3>
                <div class="details-grid">
                    <div class="detail-item"><strong>Brand:</strong> ${escapeHtml(instance.CPUInfo.Brand)}</div>
                    <div class="detail-item"><strong>Model:</strong> ${escapeHtml(instance.CPUInfo.Model)}</div>
                    <div class="detail-item"><strong>Cores:</strong> ${instance.CPUInfo.Cores}</div>
                    <div class="detail-item"><strong>Threads:</strong> ${instance.CPUInfo.Threads}</div>
                    <div class="detail-item"><strong>Speed:</strong> ${instance.CPUInfo.SpeedGHz} GHz</div>
                    <div class="detail-item"><strong>Cache Size:</strong> ${instance.CPUInfo.CacheSize} MB</div>
                    <div class="detail-item"><strong>Socket Count:</strong> ${instance.CPUInfo.SocketCount}</div>
                </div>
            </div>
        `;
    }
    
    if (instance.RAMInfo) {
        html += `
            <div class="details-section">
                <h3>Memory (RAM)</h3>
                <div class="details-grid">
                    <div class="detail-item"><strong>Total:</strong> ${instance.RAMInfo.TotalGB} GB</div>
                    <div class="detail-item"><strong>Configuration:</strong> ${escapeHtml(instance.RAMInfo.Configuration)}</div>
                    <div class="detail-item"><strong>Type:</strong> ${escapeHtml(instance.RAMInfo.Type)}</div>
                    <div class="detail-item"><strong>Speed:</strong> ${instance.RAMInfo.Speed} MHz</div>
                    <div class="detail-item"><strong>ECC:</strong> ${instance.RAMInfo.ECC ? 'Yes' : 'No'}</div>
                    <div class="detail-item"><strong>Manufacturer:</strong> ${escapeHtml(instance.RAMInfo.Manufacturer)}</div>
                </div>
            </div>
        `;
    }
    
    if (instance.StorageDisks && instance.StorageDisks.length > 0) {
        html += `
            <div class="details-section">
                <h3>Storage Disks (${instance.StorageDisks.length})</h3>
                <div class="storage-table">
                    <table>
                        <thead>
                            <tr>
                                <th>Slot</th>
                                <th>Type</th>
                                <th>Brand/Model</th>
                                <th>Capacity</th>
                                <th>Used</th>
                                <th>Usage</th>
                                <th>Health</th>
                                <th>Temp</th>
                            </tr>
                        </thead>
                        <tbody>
        `;
        
        instance.StorageDisks.forEach(disk => {
            const healthClass = disk.HealthStatus === 'healthy' ? 'health-ok' : 'health-warning';
            html += `
                <tr>
                    <td>${disk.Slot}</td>
                    <td><span class="disk-type">${escapeHtml(disk.Type)}</span></td>
                    <td><div class="disk-model">${escapeHtml(disk.Brand)} ${escapeHtml(disk.Model)}</div><div class="disk-serial">${escapeHtml(disk.SerialNumber)}</div></td>
                    <td>${disk.CapacityGB} GB</td>
                    <td>${disk.UsedGB} GB</td>
                    <td><div class="usage-bar"><div class="usage-fill" style="width: ${disk.UsagePercent}%"></div></div><span class="usage-text">${disk.UsagePercent.toFixed(1)}%</span></td>
                    <td><span class="${healthClass}">${escapeHtml(disk.HealthStatus)}</span></td>
                    <td>${disk.TemperatureC}°C</td>
                </tr>
            `;
        });
        
        html += `
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }
    
    if (instance.NetworkNICs && instance.NetworkNICs.length > 0) {
        html += `
            <div class="details-section">
                <h3>Network Interfaces (${instance.NetworkNICs.length})</h3>
                <div class="network-table">
                    <table>
                        <thead>
                            <tr>
                                <th>Interface</th>
                                <th>Vendor/Model</th>
                                <th>IPv4</th>
                                <th>IPv6</th>
                                <th>MAC Address</th>
                                <th>Bandwidth</th>
                                <th>Status</th>
                            </tr>
                        </thead>
                        <tbody>
        `;
        
        instance.NetworkNICs.forEach(nic => {
            const statusClass = nic.Status === 'up' ? 'status-running' : 'status-stopped';
            html += `
                <tr>
                    <td><strong>${escapeHtml(nic.Interface)}</strong></td>
                    <td><div class="nic-vendor">${escapeHtml(nic.Vendor)}</div><div class="nic-model">${escapeHtml(nic.Model)}</div></td>
                    <td>${escapeHtml(nic.IPv4)}</td>
                    <td class="ipv6">${escapeHtml(nic.IPv6)}</td>
                    <td class="mac-address">${escapeHtml(nic.MACAddress)}</td>
                    <td>${nic.BandwidthGbps} Gbps</td>
                    <td><span class="status-badge ${statusClass}">● ${escapeHtml(nic.Status)}</span></td>
                </tr>
            `;
        });
        
        html += `
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }
    
    return html;
}

function closeModal() {
    const modal = document.getElementById('detailsModal');
    modal.style.display = 'none';
}

function setLoading(loading) {
    const refreshBtn = document.getElementById('refreshBtn');
    const loader = document.getElementById('loader');
    const tableContainer = document.querySelector('.table-container');
    
    if (loading) {
        refreshBtn.disabled = true;
        loader.style.display = 'block';
        if (tableContainer) tableContainer.style.opacity = '0.5';
    } else {
        refreshBtn.disabled = false;
        loader.style.display = 'none';
        if (tableContainer) tableContainer.style.opacity = '1';
    }
}

function showError(message) {
    const tbody = document.getElementById('instancesTableBody');
    tbody.innerHTML = `<tr class="empty-row"><td colspan="7"><div class="error-state">❌ Error: ${escapeHtml(message)}</div></td></tr>`;
}

async function loadServers() {
    setLoading(true);
    
    try {
        const response = await apiManager.fetchServerList();
        
        // Parse positional data format
        if (response.columns && response.data) {
            // Build column mapping
            columnMapping = {};
            response.columns.forEach((col, idx) => {
                columnMapping[col] = idx;
            });
            
            // Store raw positional arrays
            servers = response.data;
        } else {
            // Fallback for object format
            servers = response || [];
        }
        
        filteredServers = [...servers];
        updateStats();
        renderServers();
        
        const timestamp = new Date().toLocaleTimeString();
        document.getElementById('lastUpdate').textContent = timestamp;
        
        // Load filters if not already loaded
        if (!filterOptions) {
            const filters = await apiManager.fetchFilters();
            populateFilterDropdowns(filters);
        }
    } catch (error) {
        console.error('Failed to load servers:', error);
        showError(error.message);
    } finally {
        setLoading(false);
    }
}

window.goToPage = function(page) {
    currentPage = page;
    renderServers();
    window.scrollTo({ top: 0, behavior: 'smooth' });
};

// Initialize - works whether DOM is ready or not
function initialize() {
    apiManager = createAPIManager();
    
    // Set up event listeners
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadServers);
    }
    
    const searchInput = document.getElementById('searchInput');
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            filterServers(e.target.value);
        });
    }

    // Add filter dropdown listeners
    const regionFilter = document.getElementById('regionFilter');
    if (regionFilter) {
        regionFilter.addEventListener('change', (e) => {
            activeFilters.region = e.target.value;
            applyFilters();
        });
    }

    const zoneFilter = document.getElementById('zoneFilter');
    if (zoneFilter) {
        zoneFilter.addEventListener('change', (e) => {
            activeFilters.zone = e.target.value;
            applyFilters();
        });
    }

    const stateFilter = document.getElementById('stateFilter');
    if (stateFilter) {
        stateFilter.addEventListener('change', (e) => {
            activeFilters.state = e.target.value;
            applyFilters();
        });
    }

    const typeFilter = document.getElementById('typeFilter');
    if (typeFilter) {
        typeFilter.addEventListener('change', (e) => {
            activeFilters.type = e.target.value;
            applyFilters();
        });
    }
    
    // Auto-load on startup
    loadServers();
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initialize);
} else {
    // DOM already loaded
    initialize();
}

// Make closeModal available globally
window.closeModal = closeModal;
