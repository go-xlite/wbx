// Proxy Test Console JavaScript

// Extract path prefix from current URL
const pathParts = window.location.pathname.split('/').filter(p => p);
const pathPrefix = pathParts.length > 0 ? '/' + pathParts[0] : '';

// Update proxy endpoint display
document.getElementById('proxy-endpoint').textContent = `${pathPrefix}/proxy/`;

// Statistics tracking
let stats = {
    total: 0,
    successful: 0,
    failed: 0,
    lastStatus: '-'
};

// DOM elements
const pathInput = document.getElementById('path-input');
const methodSelect = document.getElementById('method-select');
const sendRequestBtn = document.getElementById('send-request-btn');
const clearBtn = document.getElementById('clear-btn');
const messageLog = document.getElementById('message-log');
const responseHeaders = document.getElementById('response-headers');
const responseBody = document.getElementById('response-body');

// Update statistics display
function updateStats() {
    document.getElementById('total-requests').textContent = stats.total;
    document.getElementById('successful-requests').textContent = stats.successful;
    document.getElementById('failed-requests').textContent = stats.failed;
    document.getElementById('last-status').textContent = stats.lastStatus;
}

// Add log entry
function addLog(message, type = 'info') {
    const time = new Date().toLocaleTimeString();
    const entry = document.createElement('div');
    entry.className = `log-entry log-${type}`;
    entry.innerHTML = `<span class="log-time">[${time}]</span>${message}`;
    messageLog.appendChild(entry);
    messageLog.scrollTop = messageLog.scrollHeight;
}

// Clear logs and response
function clearLogs() {
    messageLog.innerHTML = '';
    responseHeaders.innerHTML = '<em>Response headers will appear here...</em>';
    responseBody.innerHTML = '<em>Response body will appear here...</em>';
    addLog('Logs cleared', 'info');
}

// Send proxy request
async function sendRequest() {
    const path = pathInput.value.trim();
    const method = methodSelect.value;
    
    // Build the proxy URL
    const proxyUrl = `${pathPrefix}/proxy/${path}`;
    
    stats.total++;
    updateStats();
    
    addLog(`Sending ${method} request to: ${proxyUrl}`, 'info');
    
    try {
        const startTime = Date.now();
        const response = await fetch(proxyUrl, {
            method: method,
            headers: {
                'Accept': '*/*'
            }
        });
        
        const duration = Date.now() - startTime;
        const status = response.status;
        const statusText = response.statusText;
        
        stats.lastStatus = `${status} ${statusText}`;
        
        // Display response headers
        const headers = {};
        response.headers.forEach((value, key) => {
            headers[key] = value;
        });
        
        let headersHtml = '<strong>Response Headers:</strong><br>';
        for (const [key, value] of Object.entries(headers)) {
            headersHtml += `<div><strong>${key}:</strong> ${value}</div>`;
        }
        responseHeaders.innerHTML = headersHtml;
        
        // Get response body
        const contentType = response.headers.get('content-type') || '';
        let body;
        
        if (contentType.includes('application/json')) {
            body = await response.json();
            responseBody.textContent = JSON.stringify(body, null, 2);
        } else if (contentType.includes('text/')) {
            body = await response.text();
            responseBody.textContent = body;
        } else {
            const blob = await response.blob();
            responseBody.textContent = `Binary data (${blob.size} bytes)\nContent-Type: ${contentType}`;
        }
        
        if (response.ok) {
            stats.successful++;
            addLog(`✓ Success: ${status} ${statusText} (${duration}ms)`, 'success');
        } else {
            stats.failed++;
            addLog(`✗ Error: ${status} ${statusText} (${duration}ms)`, 'error');
        }
        
    } catch (error) {
        stats.failed++;
        stats.lastStatus = 'Error';
        addLog(`✗ Request failed: ${error.message}`, 'error');
        responseHeaders.innerHTML = '<em style="color: red;">Request failed</em>';
        responseBody.textContent = `Error: ${error.message}`;
    }
    
    updateStats();
}

// Event listeners
sendRequestBtn.addEventListener('click', sendRequest);
clearBtn.addEventListener('click', clearLogs);

pathInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        sendRequest();
    }
});

// Initialize
addLog('Proxy test console initialized', 'success');
addLog(`Target: https://file-drop.gtn.one:8080/xt21/`, 'info');
addLog(`Proxy endpoint: ${pathPrefix}/proxy/`, 'info');
updateStats();

// Set initial placeholder text
responseHeaders.innerHTML = '<em>Response headers will appear here...</em>';
responseBody.innerHTML = '<em>Response body will appear here...</em>';
