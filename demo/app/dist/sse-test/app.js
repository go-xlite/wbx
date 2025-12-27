let eventSource = null;
let receivedCount = 0;
let reconnectAttempts = 0;
let connectTime = null;
let uptimeInterval = null;
let shouldReconnect = true;
let reconnectTimeout = null;

function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function updateStatus(status) {
    const statusEl = document.getElementById('sseStatus');
    statusEl.className = 'sse-status ' + status;
    if (status === 'connected') {
        statusEl.textContent = '🟢 Connected to SSE';
    } else if (status === 'disconnected') {
        statusEl.textContent = '🔴 Disconnected';
    } else if (status === 'connecting') {
        statusEl.textContent = '🟡 Connecting...';
    }
}

function addMessage(msg, type = 'info') {
    const messagesDiv = document.getElementById('messages');
    const msgDiv = document.createElement('div');
    msgDiv.className = 'message ' + type;
    const timestamp = new Date().toLocaleTimeString();
    msgDiv.innerHTML = `<strong>[${timestamp}]</strong> ${escapeHtml(msg)}`;
    messagesDiv.appendChild(msgDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function updateStats() {
    document.getElementById('receivedCount').textContent = receivedCount;
    document.getElementById('reconnects').textContent = reconnectAttempts;
}

function updateUptime() {
    if (connectTime) {
        const elapsed = Math.floor((Date.now() - connectTime) / 1000);
        const hours = Math.floor(elapsed / 3600);
        const minutes = Math.floor((elapsed % 3600) / 60);
        const seconds = elapsed % 60;
        document.getElementById('uptime').textContent = 
            `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    } else {
        document.getElementById('uptime').textContent = '00:00:00';
    }
}

function connect() {
    if (eventSource) {
        addMessage('Already connected or connecting', 'error');
        return;
    }

    shouldReconnect = true;
    updateStatus('connecting');
    
    // Construct SSE URL based on current page location
    // Extract path prefix from current URL (e.g., /xt23 from /xt23/sse-test/)
    const pathParts = window.location.pathname.split('/').filter(p => p);
    const pathPrefix = pathParts.length > 0 ? '/' + pathParts[0] : '';
    
    const protocol = window.location.protocol;
    const sseUrl = `${protocol}//${window.location.host}${pathPrefix}/sse/stream`;
    
    // Update the displayed endpoint
    document.getElementById('sseEndpoint').textContent = sseUrl;
    
    addMessage(`Connecting to ${sseUrl}...`, 'info');

    eventSource = new EventSource(sseUrl);

    eventSource.onopen = function() {
        updateStatus('connected');
        addMessage('Connected successfully!', 'received');
        connectTime = Date.now();
        uptimeInterval = setInterval(updateUptime, 1000);
        reconnectAttempts = 0;
        updateStats();
        document.getElementById('connectBtn').disabled = true;
        document.getElementById('disconnectBtn').disabled = false;
    };

    eventSource.onmessage = function(event) {
        receivedCount++;
        updateStats();
        addMessage('Received: ' + event.data, 'received');
    };

    eventSource.onerror = function(error) {
        addMessage('SSE connection error occurred', 'error');
        console.error('SSE error:', error);
        
        // EventSource automatically tries to reconnect, but we'll track it
        if (eventSource.readyState === EventSource.CLOSED) {
            handleDisconnect();
        }
    };
}

function handleDisconnect() {
    updateStatus('disconnected');
    connectTime = null;
    if (uptimeInterval) {
        clearInterval(uptimeInterval);
        uptimeInterval = null;
    }
    updateUptime();
    document.getElementById('connectBtn').disabled = false;
    document.getElementById('disconnectBtn').disabled = true;
    
    if (shouldReconnect && eventSource) {
        reconnectAttempts++;
        updateStats();
        addMessage(`Connection lost. Reconnecting in 5 seconds... (Attempt ${reconnectAttempts})`, 'info');
        
        reconnectTimeout = setTimeout(() => {
            if (shouldReconnect) {
                eventSource = null;
                connect();
            }
        }, 5000);
    } else {
        addMessage('Connection closed', 'error');
        eventSource = null;
    }
}

function disconnect() {
    shouldReconnect = false;
    if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
        reconnectTimeout = null;
    }
    if (eventSource) {
        addMessage('Disconnecting...', 'info');
        eventSource.close();
        eventSource = null;
        handleDisconnect();
    }
}

function clearMessages() {
    document.getElementById('messages').innerHTML = '';
    addMessage('Message log cleared', 'info');
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    updateStatus('disconnected');
    updateStats();
    updateUptime();
    addMessage('SSE Test Console loaded. Click Connect to start.', 'info');
});

// Cleanup on page unload
window.addEventListener('beforeunload', function() {
    shouldReconnect = false;
    if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
    }
    if (eventSource) {
        eventSource.close();
    }
    if (uptimeInterval) {
        clearInterval(uptimeInterval);
    }
});
