let ws = null;
let sentCount = 0;
let receivedCount = 0;
let connectTime = null;
let uptimeInterval = null;

function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function updateStatus(status) {
    const statusEl = document.getElementById('wsStatus');
    statusEl.className = 'ws-status ' + status;
    if (status === 'connected') {
        statusEl.textContent = '🟢 Connected to WebSocket';
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
    document.getElementById('sentCount').textContent = sentCount;
    document.getElementById('receivedCount').textContent = receivedCount;
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
    if (ws) {
        addMessage('Already connected', 'error');
        return;
    }

    updateStatus('connecting');
    
    // Construct WebSocket URL based on current page location
    // Extract path prefix from current URL (e.g., /xt23 from /xt23/ws-test/)
    const pathParts = window.location.pathname.split('/').filter(p => p);
    const pathPrefix = pathParts.length > 0 ? '/' + pathParts[0] : '';
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}${pathPrefix}/ws/connect`;
    addMessage(`Connecting to ${wsUrl}...`, 'info');

    ws = new WebSocket(wsUrl);

    ws.onopen = function() {
        updateStatus('connected');
        addMessage('Connected successfully!', 'received');
        connectTime = Date.now();
        uptimeInterval = setInterval(updateUptime, 1000);
        document.getElementById('connectBtn').disabled = true;
        document.getElementById('disconnectBtn').disabled = false;
        document.getElementById('sendBtn').disabled = false;
        document.getElementById('pingBtn').disabled = false;
        document.getElementById('burstBtn').disabled = false;
    };

    ws.onmessage = function(event) {
        receivedCount++;
        updateStats();
        addMessage('Received: ' + event.data, 'received');
    };

    ws.onerror = function(error) {
        addMessage('WebSocket error occurred', 'error');
        console.error('WebSocket error:', error);
    };

    ws.onclose = function() {
        updateStatus('disconnected');
        addMessage('Connection closed', 'error');
        ws = null;
        connectTime = null;
        if (uptimeInterval) {
            clearInterval(uptimeInterval);
            uptimeInterval = null;
        }
        updateUptime();
        document.getElementById('connectBtn').disabled = false;
        document.getElementById('disconnectBtn').disabled = true;
        document.getElementById('sendBtn').disabled = true;
        document.getElementById('pingBtn').disabled = true;
        document.getElementById('burstBtn').disabled = true;
    };
}

function disconnect() {
    if (ws) {
        addMessage('Disconnecting...', 'info');
        ws.close();
    }
}

function sendMessage() {
    if (!ws) {
        addMessage('Not connected!', 'error');
        return;
    }

    const input = document.getElementById('messageInput');
    const message = input.value.trim();
    
    if (!message) {
        addMessage('Cannot send empty message', 'error');
        return;
    }

    ws.send(message);
    sentCount++;
    updateStats();
    addMessage('Sent: ' + message, 'sent');
    input.value = '';
}

function sendPing() {
    if (!ws) {
        addMessage('Not connected!', 'error');
        return;
    }

    const ping = JSON.stringify({ type: 'ping', timestamp: Date.now() });
    ws.send(ping);
    sentCount++;
    updateStats();
    addMessage('Sent PING', 'sent');
}

function sendBurst() {
    if (!ws) {
        addMessage('Not connected!', 'error');
        return;
    }

    for (let i = 1; i <= 10; i++) {
        const msg = `Burst message ${i}/10`;
        ws.send(msg);
        sentCount++;
    }
    updateStats();
    addMessage('Sent burst of 10 messages', 'sent');
}

function clearMessages() {
    document.getElementById('messages').innerHTML = '';
    addMessage('Message log cleared', 'info');
}

// Enter key to send
document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('messageInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            sendMessage();
        }
    });

    // Initial state
    updateStatus('disconnected');
    updateStats();
    updateUptime();
});
